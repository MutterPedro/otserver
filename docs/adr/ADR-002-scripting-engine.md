# ADR-002: Scripting Engine (Lua)

- **Status:** Accepted
- **Date:** 2026-02-23
- **Phase:** 1 — Architecture Decision Workshop

---

## Context

TFS uses Lua 5.1 extensively. Approximately 130 C API bridge functions expose server internals to scripts. All game events — creature movement, NPC dialogue, item use, spell casting, tile triggers — route through Lua callbacks registered via XML event descriptors. The scripting layer must be preserved with full source compatibility: existing `data/` Lua scripts must run without modification.

---

## Options Considered

| Option | Pros | Cons |
|--------|------|------|
| `gopher-lua` (pure Go) | No CGO; static single-binary deployment; idiomatic Go GC integration; good Lua 5.1 compatibility | Slower than LuaJIT for CPU-heavy scripts (~3–5× on tight loops) |
| `go-lua` / CGO bindings to C Lua 5.1 | Near-native execution speed | CGO context-switch penalty per call; CGO breaks Go's race detector; complicates cross-compilation and Docker builds |
| Embedded V8 / Wasm scripting | Modern, fast | Complete rewrite of all existing Lua scripts; out of scope for a compatibility port |

---

## Decision

**`github.com/yuin/gopher-lua` (pure Go Lua 5.1 implementation)** for the initial port.

The CGO approach is rejected because:

1. CGO disables Go's race detector (`-race`), which is critical during development.
2. CGO adds per-call overhead that can exceed the execution-time savings for the short-lived event callbacks typical in a game server.
3. Pure-Go builds allow `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build` — a single static binary with no `.so` dependencies.

Execution speed is acceptable: Lua callbacks in TFS fire for discrete events (item use, step-on), not in tight numeric loops. The bottleneck in a real game server is network I/O and map traversal, not Lua arithmetic.

---

## Critical Concurrency Constraint

`lua.LState` is **not goroutine-safe**. A single shared `LState` accessed from multiple goroutines will corrupt the Lua VM and cause data races or panics.

**Solution: `sync.Pool` of pre-compiled `LState` instances.**

Each `LState` in the pool is fully initialized with the server's Lua module loaded. When a goroutine needs to run a Lua event:

1. Acquire an `LState` from the pool (`pool.Get()`).
2. Execute the script callback.
3. Return the `LState` to the pool (`pool.Put(state)`).

This allows fully parallel, independent Lua events (e.g., two players stepping on different tiles simultaneously) with zero lock contention.

```
Event A ──▶  pool.Get() ──▶  LState₁ ──▶  Execute ──▶  pool.Put()
Event B ──▶  pool.Get() ──▶  LState₂ ──▶  Execute ──▶  pool.Put()
```

---

## Script Engine Interface Contract

```go
// internal/script/engine.go

// Engine is the interface to the Lua scripting layer.
// Implementations must be safe to call from multiple goroutines concurrently.
type Engine interface {
    // CallEvent dispatches a named Lua event with the given arguments.
    // It is goroutine-safe: each call acquires a dedicated LState from
    // the internal pool.
    CallEvent(ctx context.Context, event string, args ...lua.LValue) (lua.LValue, error)

    // RegisterLib loads a named Lua module into every pooled LState.
    // Must be called before the pool is first used (i.e., at startup).
    RegisterLib(name string, open lua.LGFunction)
}
```

---

## Consequences

- `github.com/yuin/gopher-lua` is already present in `go.mod` (correct version: `v1.1.1`).
- The 130+ C++ Lua API functions must be re-implemented as `gopher-lua` Go functions exposed via `RegisterLib`. This is the bulk of Phase 7 work.
- The XML event registration system from TFS (`<event type="login" script="login.lua"/>`) will be reproduced in Go, parsing `data/` XML files and wiring callbacks to the `Engine`.
- Lua scripts themselves are **not modified**; only the bridge layer changes.
