package config_test

import (
	"strings"
	"testing"

	"github.com/MutterPedro/otserver/internal/config"
)

func TestConfig_LoadAndParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expected    config.Config
		expectError bool
	}{
		{
			name: "valid minimal config",
			input: `
[server]
address = "0.0.0.0:7172"
max_connections = 100

[database]
dsn = "root:password@tcp(localhost:3306)/otserver"

[log]
level = "info"
`,
			expected: config.Config{
				Server: config.ServerConfig{
					Address:        "0.0.0.0:7172",
					MaxConnections: 100,
				},
				Database: config.DatabaseConfig{
					DSN: "root:password@tcp(localhost:3306)/otserver",
				},
				Log: config.LogConfig{
					Level: "info",
				},
			},
		},
		{
			name: "default max_connections when zero",
			input: `
[server]
address = "127.0.0.1:7172"

[database]
dsn = "root:@tcp(localhost:3306)/ot"

[log]
level = "warn"
`,
			expected: config.Config{
				Server: config.ServerConfig{
					Address:        "127.0.0.1:7172",
					MaxConnections: config.DefaultMaxConnections,
				},
				Database: config.DatabaseConfig{
					DSN: "root:@tcp(localhost:3306)/ot",
				},
				Log: config.LogConfig{
					Level: "warn",
				},
			},
		},
		{
			name:        "empty input",
			input:       ``,
			expectError: true,
		},
		{
			name:        "invalid TOML syntax",
			input:       `[server\naddress = `,
			expectError: true,
		},
		{
			name: "all valid log levels accepted",
			input: `
[server]
address = "0.0.0.0:7172"
[database]
dsn = "x"
[log]
level = "debug"
`,
			expected: config.Config{
				Server:   config.ServerConfig{Address: "0.0.0.0:7172", MaxConnections: config.DefaultMaxConnections},
				Database: config.DatabaseConfig{DSN: "x"},
				Log:      config.LogConfig{Level: "debug"},
			},
		},
		{
			name: "negative max_connections treated as default",
			input: `
[server]
address = "0.0.0.0:7172"
max_connections = -5
[database]
dsn = "x"
[log]
level = "info"
`,
			expected: config.Config{
				Server:   config.ServerConfig{Address: "0.0.0.0:7172", MaxConnections: config.DefaultMaxConnections},
				Database: config.DatabaseConfig{DSN: "x"},
				Log:      config.LogConfig{Level: "info"},
			},
		},
		{
			name: "missing required server address",
			input: `
[server]

[database]
dsn = "root:@tcp(localhost)/ot"

[log]
level = "info"
`,
			expectError: true,
		},
		{
			name: "missing required database DSN",
			input: `
[server]
address = "0.0.0.0:7172"

[database]

[log]
level = "info"
`,
			expectError: true,
		},
		{
			name: "invalid log level",
			input: `
[server]
address = "0.0.0.0:7172"

[database]
dsn = "root:@tcp(localhost)/ot"

[log]
level = "verbose"
`,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := config.LoadFromReader(strings.NewReader(tc.input))

			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error but got nil; parsed config: %+v", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Server.Address != tc.expected.Server.Address {
				t.Errorf("Server.Address = %q, want %q", got.Server.Address, tc.expected.Server.Address)
			}
			if got.Server.MaxConnections != tc.expected.Server.MaxConnections {
				t.Errorf("Server.MaxConnections = %d, want %d", got.Server.MaxConnections, tc.expected.Server.MaxConnections)
			}
			if got.Database.DSN != tc.expected.Database.DSN {
				t.Errorf("Database.DSN = %q, want %q", got.Database.DSN, tc.expected.Database.DSN)
			}
			if got.Log.Level != tc.expected.Log.Level {
				t.Errorf("Log.Level = %q, want %q", got.Log.Level, tc.expected.Log.Level)
			}
		})
	}
}
