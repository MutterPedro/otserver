// Package script defines the Lua scripting engine interface for the OTServer.
// The concrete implementation uses gopher-lua with a sync.Pool of pre-compiled
// LState instances to allow safe concurrent script execution.
package script

import (
	"context"

	lua "github.com/yuin/gopher-lua"
)

// Engine is the server-facing interface to the Lua scripting layer.
// All methods are safe to call from multiple goroutines concurrently;
// each call acquires a dedicated lua.LState from the internal pool.
type Engine interface {
	// CallEvent dispatches a named Lua event (e.g., "onLogin", "onUseItem")
	// with the given arguments and returns the script's return value.
	// If the named event is not registered the call is a no-op and nil is returned.
	CallEvent(ctx context.Context, event string, args ...lua.LValue) (lua.LValue, error)

	// RegisterLib makes a named Lua module available inside every pooled LState.
	// It must be called before the Engine begins serving events (i.e., at startup,
	// before any goroutine calls CallEvent).
	RegisterLib(name string, open lua.LGFunction)

	// Close shuts down the engine and releases all pooled LState resources.
	Close() error
}
