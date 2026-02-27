// Package repository defines the persistence interfaces for the OTServer.
// Concrete implementations backed by PostgreSQL and Redis live in this package;
// all game-layer code depends only on these interfaces, never on driver types.
package repository

import (
	"context"
	"time"
)

// PlayerRepository persists and loads player state to/from durable storage.
type PlayerRepository interface {
	SaveState(ctx context.Context, p *PlayerState) error
	LoadState(ctx context.Context, id uint32) (*PlayerState, error)
}

// AccountRepository provides read access to account records.
type AccountRepository interface {
	FindByName(ctx context.Context, name string) (*Account, error)
	FindByID(ctx context.Context, id uint32) (*Account, error)
}

// SessionRepository manages short-lived login session tokens in Redis.
// All methods are safe to call from multiple goroutines concurrently.
type SessionRepository interface {
	Set(ctx context.Context, token string, accountID uint32, ttl time.Duration) error
	// Get retrieves the accountID associated with token.
	// Returns ErrNotFound if the token is absent or expired.
	Get(ctx context.Context, token string) (uint32, error)
	Delete(ctx context.Context, token string) error
}

// PlayerState is the serialisable snapshot of a player saved to PostgreSQL.
// Fields are intentionally flat to map cleanly to a single table row + JSONB column.
type PlayerState struct {
	ID   uint32
	Name string
}

// Account represents an authenticated user account.
type Account struct {
	ID       uint32
	Name     string
	Password string // bcrypt
}
