# ADR-006: Configuration Format & Hot-Reload

- **Status:** Accepted
- **Date:** 2026-02-23
- **Phase:** 1 — Architecture Decision Workshop

---

## Context

TFS evaluates `config.lua` at startup to load server parameters (experience rates, PvP settings, world type, etc.). Using Lua as a config format couples the configuration parser to the Lua VM and makes hot-reload difficult — the entire Lua state must be re-executed.

The Go port must provide:

1. A **human-readable, type-safe** config format.
2. **Hot-reload** — operators can change exp rates or toggle features without restarting the server.
3. **Race-free reads** — hundreds of goroutines (game loop, player sessions, Lua scripts) read config values concurrently. A hot-reload cannot introduce a data race.

---

## Options Considered

| Option | Pros | Cons |
|--------|------|------|
| `config.lua` (keep) | No migration; operators familiar with it | Lua VM required just for config; hard to hot-reload safely |
| JSON | Standard, tooling everywhere | No comments; unfriendly for human editors |
| YAML | Human-friendly; comments supported | Significant Go YAML library pitfalls (implicit type coercion, Norway problem) |
| **TOML** | Human-friendly; strong types; comments; excellent Go support | Less familiar than JSON to some; no anchors/aliases |

---

## Decision

**TOML (`github.com/pelletier/go-toml/v2`) with `github.com/fsnotify/fsnotify` for hot-reload, protected by `atomic.Pointer[Config]`.**

### Format

TOML was already chosen during Phase 0 scaffolding (see `config.toml` in the repository root and the existing `internal/config/config.go`). This ADR ratifies and extends that decision.

### Hot-Reload Mechanism

`fsnotify` watches `config.toml` for `WRITE` events. On change:

1. The watcher goroutine re-parses the file into a fresh `*Config` value.
2. It validates the new config (same rules as startup validation).
3. On success, it atomically swaps the global pointer: `configPtr.Store(newCfg)`.

All readers call `configPtr.Load()` to get the current pointer — a single atomic load with no lock and no allocation.

```
fsnotify goroutine
    │  file WRITE event
    ├─ parse + validate ──▶ *Config (new)
    └─ atomic.Pointer.Store(new)

Any goroutine
    └─ cfg := configPtr.Load()   // always consistent; never races
```

**Go Pitfall:** Using a plain global `var Config` protected by `sync.RWMutex` is error-prone — any goroutine that forgets to acquire the lock causes a data race that the race detector may not catch in every test run. `atomic.Pointer[T]` makes the correct pattern the only possible pattern.

---

## Configuration Interface Contract

```go
// internal/config/config.go (extends existing file)

// Loader manages configuration loading and hot-reloading.
type Loader interface {
    // Current returns the active configuration snapshot.
    // Safe to call from any goroutine without locking.
    Current() *Config

    // Watch starts the fsnotify watcher goroutine.
    // It stops when ctx is cancelled.
    Watch(ctx context.Context, path string) error
}
```

The existing `LoadFromReader` function remains as the pure parsing primitive used by both initial load and the watcher goroutine.

---

## Consequences

- `github.com/pelletier/go-toml/v2` and `github.com/fsnotify/fsnotify` are already present in `go.mod`.
- The `DatabaseConfig.DSN` field (generic string) will be replaced with explicit `PostgresDSN string` and `RedisAddr string` fields per ADR-001.
- `config.lua` from existing `data/` directories is superseded by `config.toml`. A one-time migration script (out of scope for this phase) can assist operators.
- Phase 0 acceptance tests (`internal/config/config_test.go`) already cover round-trip parsing. Phase 1 adds a test for the `atomic.Pointer` hot-swap behaviour.
