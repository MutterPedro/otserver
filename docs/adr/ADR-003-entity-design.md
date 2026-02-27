# ADR-003: Entity Design Pattern & Domain Modeling

- **Status:** Accepted
- **Date:** 2026-02-23
- **Phase:** 1 — Architecture Decision Workshop

---

## Context

TFS uses a deep object-oriented inheritance hierarchy rooted at `Thing`:

```
Thing
├── Item
│   ├── Container
│   └── Teleport
└── Creature
    ├── Player
    ├── Monster
    └── NPC
```

This hierarchy in C++ gives `virtual` dispatch and shared base state, but it couples all behaviour into a handful of large classes. Go has no inheritance; the canonical approach is composition and interfaces. The entity design directly affects:

- GC pressure (pointer indirection depth)
- Testability (mocking surfaces via interfaces)
- Map storage (100,000+ `Item` values held in tile lists)

---

## Options Considered

| Option | Pros | Cons |
|--------|------|------|
| Deep interface embedding (mirror C++) | Familiar mapping from C++ hierarchy; minimal rethinking | Creates large "God" interfaces; forces all entities through dynamic dispatch; GC must track every pointer indirection; impossible to unit-test sub-behaviours in isolation |
| Entity-Component-System (ECS) | Cache-friendly; highly parallelisable; decoupled behaviour | Massive logic rewrite far beyond a port; wrong scope for Phase 1 |
| **Composition with small interfaces** | Idiomatic Go; decoupled behaviour; small, mockable surfaces; minimal allocations | Requires delegation boilerplate; some patterns need explicit type switches |

---

## Decision

**Composition over inheritance with small, focused interfaces.**

Structs embed reusable components (e.g., `PositionComponent`, `HealthComponent`). Behaviour is exposed through narrow interfaces that each describe a single capability. This:

- Keeps interface surfaces small and mockable (one method is enough for a test double).
- Avoids boxing 100,000 `Item` values into `interface{}` slots on every tile — concrete item slices can be stored directly.
- Allows goroutines to operate on disjoint entity sets without shared locks (each goroutine owns its spatial partition).

---

## Domain Interface Contracts

```go
// internal/game/entity.go

// Positioner is any entity that occupies a map location.
type Positioner interface {
    Position() Position
}

// Mover can change its map location.
type Mover interface {
    Positioner
    SetPosition(pos Position)
}

// Damagable can receive damage and die.
type Damagable interface {
    TakeDamage(amount int, source DamageSource)
    Health() int
    MaxHealth() int
    IsDead() bool
}

// Creature is a living entity on the map.
type Creature interface {
    Mover
    Damagable
    ID() uint32
    Name() string
}

// Item represents a stackable or unique object on a tile.
type Item interface {
    Positioner
    ItemID() uint16
    Count() uint8
}
```

---

## Concrete Struct Layout

```go
// Shared components — embedded by value, not pointer, to reduce GC scanning.

type PositionComponent struct{ pos Position }
func (c *PositionComponent) Position() Position     { return c.pos }
func (c *PositionComponent) SetPosition(p Position) { c.pos = p }

type HealthComponent struct{ hp, maxHP int }
func (c *HealthComponent) Health()    int { return c.hp }
func (c *HealthComponent) MaxHealth() int { return c.maxHP }
func (c *HealthComponent) IsDead()   bool { return c.hp <= 0 }

// Player composes components and satisfies Creature.
type Player struct {
    PositionComponent
    HealthComponent
    id   uint32
    name string
    // … inventory, skills, etc.
}
```

---

## Consequences

- Entity types will live under `internal/game/`.
- Phase 4A writes acceptance tests against the `Creature` and `Item` interfaces before any concrete struct is implemented.
- The `Damagable` interface (not a concrete `Player`) is what combat logic accepts, enabling full unit testing without a real player.
- Map tile storage uses `[]Item` (concrete slice) rather than `[]interface{}`, halving GC pressure for large maps.
