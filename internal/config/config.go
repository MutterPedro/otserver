// Package config implements configuration loading and validation for the OTS
// server. It parses TOML configuration files and provides default values for
// missing fields.
package config

import (
	"fmt"
	"io"

	"github.com/pelletier/go-toml/v2"
)

const DefaultMaxConnections = 1000

type ServerConfig struct {
	Address        string `toml:"address"`
	MaxConnections int    `toml:"max_connections"`
}

type DatabaseConfig struct {
	DSN string `toml:"dsn"`
}
type LogConfig struct {
	Level string `toml:"level"`
}

type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	Log      LogConfig      `toml:"log"`
}

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
	default:
		return Config{}, fmt.Errorf("log.level must be one of: debug, info, warn, error; got %q", cfg.Log.Level)
	}

	if cfg.Server.MaxConnections <= 0 {
		cfg.Server.MaxConnections = DefaultMaxConnections
	}

	return cfg, nil
}
