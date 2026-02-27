# ADR-005: Map Storage Format

- **Status:** Accepted
- **Date:** 2026-02-23
- **Phase:** 1 — Architecture Decision Workshop

---

## Context

TFS loads world maps from `.otbm` (Open Tibia Binary Map) files, a custom binary tree format. A full Tibia map can exceed 100 MB. The C++ server reads the entire file synchronously in the main thread at startup, which:

1. Blocks the server from accepting connections for several seconds.
2. Causes a large initial heap allocation, triggering GC pressure on the Go runtime.

The `.otbm` format must remain unchanged — it is the universal exchange format for TFS-compatible map editors (Remere's Map Editor, etc.). Changing it would break the entire tool ecosystem.

---

## Options Considered

| Option | Pros | Cons |
|--------|------|------|
| Synchronous single-thread load (mirror C++) | Simplest implementation | Server unavailable during load; large heap spike at startup triggers aggressive GC |
| Convert to SQLite / DB format | Fast partial queries | Breaks all existing map editors; migration tooling required |
| **`mmap` + concurrent chunk workers** | Avoids large GC-visible heap allocation; overlaps I/O with parsing; server can accept (and queue) connections while map loads | Slightly more complex; `mmap` requires `syscall` or `golang.org/x/sys` |

---

## Decision

**Retain `.otbm` as the authoritative map format. Load it with `mmap` and dispatch chunk-parsing to a bounded worker pool.**

### Loading Strategy

1. **`mmap` the file** at startup. The OS virtual memory system handles paging; the Go heap sees only a `[]byte` slice header (24 bytes), not the full 100 MB allocation. GC pressure is near zero.
2. **Parse the OTBM node tree header** in the main loader goroutine to discover chunk boundaries (OTBM is a tree of nodes, each with a known size).
3. **Dispatch chunk goroutines** — one per map area node — via a `semaphore`-bounded worker pool (default: `runtime.NumCPU()` workers). Each worker parses tiles, items, and spawn data independently.
4. **Barrier**: The server signals "map ready" only after all workers complete (via `sync.WaitGroup`). During loading, new connections are accepted but held in a pre-game queue.

```
startup
  │
  ├─ mmap(world.otbm) ──▶ []byte (OS-managed pages)
  │
  ├─ parse root node ──▶ chunk boundaries []ChunkRef
  │
  ├─ worker pool ─┬─ goroutine: parse chunk[0..N/4]
  │               ├─ goroutine: parse chunk[N/4..N/2]
  │               ├─ goroutine: parse chunk[N/2..3N/4]
  │               └─ goroutine: parse chunk[3N/4..N]
  │
  └─ WaitGroup.Wait() ──▶ map ready
```

---

## Map Loader Interface Contract

```go
// internal/map/loader.go (stub)

// Loader reads an OTBM file and populates the world map.
type Loader interface {
    // Load parses the OTBM file at path and returns a fully populated Map.
    // It must be safe to call with ctx cancelled (returns early if cancelled).
    Load(ctx context.Context, path string) (*Map, error)
}

// Map is the in-memory representation of the world.
type Map struct {
    Width, Height uint16
    // Tiles is indexed by [x][y][z] using a flat slice for cache locality.
    Tiles []Tile
}
```

---

## Consequences

- Requires `golang.org/x/sys` (already transitively present via other dependencies) or `syscall.Mmap` for the memory map.
- Phase 2C will implement the `Loader` behind this interface, with acceptance tests that load a small synthetic `.otbm` fixture and assert tile/item counts.
- The `.otbm` parser itself is stateless per-chunk, enabling easy unit testing of individual node parsers.
- Future hot-reload of map patches (e.g., live map editing) can re-use the same chunk-dispatch model for partial reloads.
