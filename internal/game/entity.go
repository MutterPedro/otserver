// Package game defines the core domain types and interfaces for the OTServer.
// All game-layer logic (combat, movement, scripting) depends on these interfaces,
// not on concrete struct types, to enable testing with lightweight doubles.
package game

// Position is a three-dimensional map coordinate.
type Position struct {
	X, Y uint16
	Z    uint8
}

// DamageSource describes the origin of an incoming damage event.
type DamageSource struct {
	AttackerID uint32
	Type       DamageType
}

// DamageType enumerates the kinds of damage a creature can receive.
type DamageType uint8

const (
	DamagePhysical DamageType = iota
	DamageFire
	DamageIce
	DamageEnergy
	DamagePoison
	DamageLifeDrain
	DamageManarain
	DamageHealing
)

// Positioner is any entity that occupies a location on the map.
type Positioner interface {
	Position() Position
}

// Mover is an entity that can change its map location.
type Mover interface {
	Positioner
	SetPosition(pos Position)
}

// Damagable is an entity that can receive damage and eventually die.
type Damagable interface {
	TakeDamage(amount int, source DamageSource)
	Health() int
	MaxHealth() int
	IsDead() bool
}

// Creature is a living, movable entity on the map (Player, Monster, or NPC).
type Creature interface {
	Mover
	Damagable
	ID() uint32
	Name() string
}

// Item represents a stackable or unique object placed on a tile.
type Item interface {
	Positioner
	ItemID() uint16
	Count() uint8
}
