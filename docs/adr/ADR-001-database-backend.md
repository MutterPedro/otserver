# ADR-001: Database Backend & State Persistence

- **Status:** Accepted
- **Date:** 2026-02-23
- **Phase:** 1 — Architecture Decision Workshop

---

## Context

The original TFS C++ server uses MySQL/MariaDB with synchronous, blocking queries executed directly inside game-loop handlers. This creates a hard bottleneck: every database round-trip stalls the goroutine (and in the C++ case, the thread) that called it, directly limiting the server's throughput.

A Tibia server has three distinct data access patterns that have very different consistency and latency requirements:

| Pattern | Examples | Requirement |
|---------|----------|-------------|
| Relational / schema-strict | Accounts, Characters, Guild memberships | Strong consistency, foreign-key integrity |
| Semi-structured / blob | Item attributes, quest flags, storage keys | Flexible schema, nested structure |
| Ephemeral / high-frequency | Login sessions, rate limiting, online player list | Sub-millisecond reads, TTL-based expiry |

No single database technology handles all three optimally.

---

## Options Considered

| Option | Pros | Cons |
|--------|------|------|
| MySQL (keep) | Schema reuse from C++ port, familiar | Blocking driver penalty, relational impedance for complex item hierarchies, no built-in JSONB |
| MongoDB | Natural fit for nested item/storage documents | Eventual consistency model; loses strict schema for accounts/players; weaker Go driver ecosystem vs pgx |
| PostgreSQL + Redis | Relational strictness + JSONB on one side, sub-ms ephemeral state on the other | Two services to operate instead of one |

---

## Decision

**PostgreSQL (`jackc/pgx/v5` via `pgxpool`) + Redis (`redis/go-redis/v9`) — accessed through the Repository Pattern.**

PostgreSQL is chosen over MySQL because:

- `pgxpool` provides non-blocking connection pooling that integrates natively with Go contexts and the `pgx` wire protocol, avoiding the `database/sql` abstraction overhead.
- `JSONB` columns handle arbitrary item attribute maps (`{charges: 5, enchant: "life"}`) without a separate key-value table.
- `COPY FROM` / `unnest` bulk upserts are dramatically faster than MySQL's `INSERT … ON DUPLICATE KEY UPDATE` for batch player saves.

Redis is added to handle all ephemeral operational data:

- Online player list (set membership, O(1) lookup).
- Login session tokens with automatic TTL expiry.
- Rate limiting counters per IP.
- Pub/Sub for future multi-server broadcast (sharding).

### Persistence Architecture

Saves from the game loop are **never synchronous**. The game loop dispatches save events over a buffered Go channel to a pool of background worker goroutines that batch-upsert to PostgreSQL.

```
Game Loop  ──channel──▶  Save Worker Pool  ──batch upsert──▶  PostgreSQL
    │
    └─── fast read/write ──▶  Redis
```

---

## Repository Interface Contract

```go
// internal/repository/repository.go

type PlayerRepository interface {
    SaveState(ctx context.Context, p *game.Player) error
    LoadState(ctx context.Context, id uint32) (*game.Player, error)
}

type AccountRepository interface {
    FindByName(ctx context.Context, name string) (*game.Account, error)
    FindByID(ctx context.Context, id uint32) (*game.Account, error)
}

type SessionRepository interface {
    Set(ctx context.Context, token string, accountID uint32, ttl time.Duration) error
    Get(ctx context.Context, token string) (uint32, error)
    Delete(ctx context.Context, token string) error
}
```

---

## Consequences

- **go.mod** must include `github.com/jackc/pgx/v5` and `github.com/redis/go-redis/v9`. The `github.com/go-sql-driver/mysql` dependency is removed.
- Phase 6 will implement concrete `pgxPlayerRepository` and `redisSessionRepository` types behind these interfaces.
- Tests for Phase 6 use `testcontainers-go` to spin up real PostgreSQL and Redis instances, never mocks for persistence tests.
- The existing `DatabaseConfig.DSN` field in `internal/config/config.go` will be split into separate `PostgresDSN` and `RedisAddr` fields.
