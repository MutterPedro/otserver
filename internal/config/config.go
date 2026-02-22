package config

import (
	"fmt"
	"io"

	"github.com/pelletier/go-toml/v2"
)

// DefaultMaxConnections is the fallback when max_connections is zero or negative.
const DefaultMaxConnections = 1000

// ServerConfig holds TCP server configuration.
type ServerConfig struct {
	Address        string `toml:"address"`
	MaxConnections int    `toml:"max_connections"`
}

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	DSN string `toml:"dsn"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level string `toml:"level"`
}

// Config is the top-level application configuration.
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	Log      LogConfig      `toml:"log"`
}

// LoadFromReader parses TOML from r and returns a validated Config.
func LoadFromReader(r io.Reader) (Config, error) {
	var cfg Config

	decoder := toml.NewDecoder(r)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse TOML: %w", err)
	}

	if cfg.Server.Address == "" {
		return Config{}, fmt.Errorf("server.address is required")
	}

	if cfg.Database.DSN == "" {
		return Config{}, fmt.Errorf("database.dsn is required")
	}

	switch cfg.Log.Level {
	case "debug", "info", "warn", "error":
		// valid
	default:
		return Config{}, fmt.Errorf("log.level must be one of: debug, info, warn, error; got %q", cfg.Log.Level)
	}

	if cfg.Server.MaxConnections <= 0 {
		cfg.Server.MaxConnections = DefaultMaxConnections
	}

	return cfg, nil
}
