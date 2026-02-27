# ADR-004: Network & Concurrency Model

- **Status:** Accepted
- **Date:** 2026-02-23
- **Phase:** 1 — Architecture Decision Workshop

---

## Context

TFS relies on Boost.Asio / epoll non-blocking I/O with a thread pool to handle concurrent player connections. In Go, the runtime scheduler and `net` package already provide equivalent (or superior) multiplexed I/O without requiring manual epoll management. The design must handle:

- 500–2,000 concurrent TCP connections (typical OT server load).
- Binary Tibia protocol: length-prefixed packets with XTEA encryption and Adler32 checksums.
- A deterministic, tick-based game loop that processes movement, combat, and map events at a fixed rate (50 ms / 20 Hz).
- Slow or unresponsive clients that could otherwise stall sends.

---

## Decision

**Goroutine-per-Connection (read + write goroutine pair) + central Tick-based Game Loop goroutine.**

### Per-Connection Model

Each accepted TCP connection spawns exactly two goroutines:

| Goroutine | Responsibility |
|-----------|---------------|
| **Read loop** | Reads raw bytes, decrypts XTEA, parses packet opcode, pushes `GameEvent` onto the game loop's inbound channel |
| **Write loop** | Receives `NetworkPacket` values from a buffered channel, encrypts XTEA, frames with length prefix and Adler32, writes to the TCP socket |

This eliminates any shared-state concern between the read side and write side of a connection.

### Game Loop

A single `time.Ticker` at 50 ms drives the authoritative game state. Each tick:

1. Drains the inbound `GameEvent` channel (all player actions received since last tick).
2. Advances NPC/monster AI.
3. Resolves combat, movement, and map interactions.
4. Fans out `NetworkPacket` values to each affected connection's write channel.

The game loop is the **sole writer** of game state, eliminating the need for fine-grained locks on entity state.

```
TCP Listener
    │ Accept()
    ├──▶ Read Goroutine ──GameEvent──▶ [inbound chan] ──▶ Game Loop (50ms tick)
    └──▶ Write Goroutine ◀──NetworkPacket── [outbound chan] ◀──────────────────┘
```

### Slow-Client Safety

**Edge case:** A client that reads slowly causes its outbound `chan NetworkPacket` to fill up. Without a safeguard, the game loop's non-blocking fan-out would drop packets silently; a blocking send would stall the entire tick.

**Solution:** Buffered outbound channel + non-blocking send with `select`:

```go
select {
case conn.outbound <- pkt:
    // delivered
default:
    // buffer full — client is too slow; kick it
    conn.kick("send buffer overflow")
}
```

The buffer size (default: 64 packets) is tunable in config. A client that cannot drain 64 queued packets in one tick interval is disconnected to prevent unbounded memory growth.

---

## Connection Interface Contract

```go
// internal/network/connection.go (stub)

// Connection represents an active player TCP session.
type Connection interface {
    // Send enqueues a packet for delivery to the client.
    // Returns ErrBufferFull if the outbound channel is saturated.
    Send(pkt Packet) error

    // Close terminates the connection and releases resources.
    Close(reason string)

    // Done returns a channel that is closed when the connection is torn down.
    Done() <-chan struct{}
}
```

---

## Consequences

- The existing `internal/network/Server` skeleton already follows this model (goroutine-per-conn, context-driven shutdown). Phase 3 will extend it with the full packet parsing pipeline.
- The XTEA encryption and Adler32 framing logic (partially in `internal/network/packet.go`) slot directly into the read/write goroutines.
- The game loop goroutine must never block on I/O; all persistence is delegated to background workers per ADR-001.
- Phase 3 acceptance tests drive a real TCP client against a live server goroutine — no mocks for the network layer.
