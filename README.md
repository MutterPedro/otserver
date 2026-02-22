# otserver

[![Go Report Card](https://goreportcard.com/badge/github.com/MutterPedro/otserver)](https://goreportcard.com/report/github.com/MutterPedro/otserver)
[![License](https://img.shields.io/badge/License-GPL%20v2-blue.svg)](LICENSE.md)
[![Build Status](https://github.com/MutterPedro/otserver/actions/workflows/ci.yml/badge.svg)](https://github.com/MutterPedro/otserver/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/MutterPedro/otserver.svg)](https://pkg.go.dev/github.com/MutterPedro/otserver)

A Go port of [The Forgotten Server](https://github.com/otland/forgottenserver) — a free, open-source MMORPG server emulator for the [Tibia](https://www.tibia.com) protocol. The original project is a ~50 000-line C++23 codebase; this rewrite targets Go to leverage native goroutine concurrency, simpler cross-platform deployment, and modern tooling, while maintaining **100% binary wire-protocol compatibility** with existing Tibia and OTClient game clients (same XTEA encryption, RSA key exchange, Adler32/sequence checksums, and binary packet opcodes — no client changes required).

---

## Requirements

| Tool | Version | Purpose |
| ------ | --------- | --------- |
| [Go](https://go.dev/dl/) | 1.26+ | Compiler and toolchain |
| [golangci-lint](https://golangci-lint.run/welcome/install/) | 2.x | Static analysis |
| [gofumpt](https://github.com/mvdan/gofumpt) | latest | Strict formatter |
| [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) | latest | Vulnerability scanner |
| [gotestsum](https://github.com/gotestyourself/gotestsum) | latest | Readable test output |

---

## Setup

### 1. Install Go

Download and install Go 1.26+ from the official site:

```bash
# macOS (Homebrew)
brew install go

# Linux — download the tarball from https://go.dev/dl/ then:
tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
source ~/.profile
```

Verify:

```bash
go version
# go version go1.26.0 darwin/arm64
```

### 2. Install toolchain utilities

```bash
# golangci-lint (macOS)
brew install golangci-lint

# golangci-lint (Linux/Windows — see https://golangci-lint.run/welcome/install/)
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Other tools via go install
go install mvdan.cc/gofumpt@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install gotest.tools/gotestsum@latest
```

Make sure `$(go env GOPATH)/bin` is on your `$PATH`:

```bash
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.profile
source ~/.profile
```

### 3. Clone and install dependencies

```bash
git clone https://github.com/MutterPedro/otserver.git
cd otserver
go mod download
```

### 4. Create a config file

Copy the example below to `config.toml` in the project root and adjust as needed:

```toml
[server]
address = "0.0.0.0:7172"
max_connections = 1000

[database]
dsn = "root:password@tcp(localhost:3306)/otserver"

[log]
level = "info"   # debug | info | warn | error
```

### 5. Build and run

```bash
make build
./server --config config.toml
```

---

## Development

### Makefile targets

```bash
make build       # compile binary → ./server
make test        # run all tests with verbose output
make test-race   # run all tests with the race detector (required before merging)
make test-acc    # run acceptance tests only (prefix: TestAcceptance_)
make bench       # run benchmarks with memory allocation stats
make lint        # golangci-lint with strict config
make fmt         # format all Go files with gofumpt
make vuln        # scan dependencies for known CVEs via govulncheck
```

### Running tests

```bash
# Full suite + race detector
go test -race ./...

# A specific package
go test -v ./internal/network/...

# Acceptance tests only
go test -run ^TestAcceptance ./...

# Benchmarks
go test -bench . -benchmem ./internal/crypto/...

# Fuzz testing (runs until cancelled or a panic is found)
go test -fuzz FuzzPacketDecoder ./internal/network/...
```

### Project layout

```text
otserver/
├── cmd/
│   └── server/
│       └── main.go          # Entry point: flags, DI wiring, OS signal handling
├── internal/
│   ├── config/              # TOML config loading and validation
│   ├── crypto/              # XTEA, RSA, Adler32 — pure, stateless functions
│   ├── network/             # TCP Server, Connection, packet framing
│   ├── protocol/            # Tibia packet encode/decode (opcodes, checksums)
│   ├── core/
│   │   ├── entity/          # Thing, Item, Creature, Player, Monster, NPC
│   │   ├── combat/          # Damage formulas, conditions, resolution
│   │   └── world/           # Map, Tile, Quadtree, pathfinding
│   ├── engine/              # Game loop, goroutine orchestration, dispatchers
│   ├── scripting/           # Lua 5.1 embedding via gopher-lua
│   ├── storage/             # database/sql wrappers, async persistence queue
│   └── iomap/               # OTBM map loader
├── pkg/
│   └── otb/                 # OTB item-format parser (importable as a library)
├── data/                    # Lua scripts, XML configs, map files (reused from TFS)
├── go.mod
├── go.sum
├── Makefile
├── .golangci.yml            # Strict linter config
└── .github/workflows/ci.yml # GitHub Actions: test-race + lint on every push/PR
```

### Code conventions

- Package names: lowercase, single-word (e.g. `protocol`, not `ProtocolGame`)
- No stuttering: `game.Server`, not `game.GameServer`
- Interfaces defined at the point of use (consumer-owned), not at the implementation
- All exported symbols must have doc comments (`revive` enforces this)
- Every error return must be handled or explicitly discarded with `_` (`errcheck` enforces this)

### CI

GitHub Actions runs on every push and pull request to `main`:

1. `go vet ./...`
2. `go test -race -coverprofile=coverage.out ./...`
3. `golangci-lint run ./...`
4. `govulncheck ./...`

---

## Docker

The image is built in two stages: a `golang:1.26-bookworm` builder compiles a fully-static binary (`CGO_ENABLED=0`), which is then copied into a [Google distroless](https://github.com/GoogleContainerTools/distroless) runtime image (`distroless/static-debian12`). The final image contains only the binary, CA certificates, and timezone data — no shell, no package manager.

### Build the image

```bash
docker build -t otserver:latest .
```

### Run

Mount your `config.toml` into the container and expose the game port:

```bash
docker run --rm \
  -p 7172:7172 \
  -v $(pwd)/config.toml:/etc/otserver/config.toml:ro \
  otserver:latest
```

Pass a different config path with `--config`:

```bash
docker run --rm \
  -p 7172:7172 \
  -v /srv/otserver/config.toml:/srv/config.toml:ro \
  otserver:latest --config /srv/config.toml
```

### Multi-platform builds

Use Docker Buildx to produce images for `linux/amd64` and `linux/arm64` in one step:

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t otserver:latest \
  --push .
```

---

## Tibia protocol compatibility

The server speaks the same binary protocol as the original C++ TFS, meaning any standard Tibia client or OTClient build connects without modification. Key compatibility components:

| Component | Description |
| ----------- | ------------- |
| XTEA | Symmetric stream cipher for in-game packet encryption |
| RSA | Asymmetric key exchange during login handshake |
| Adler32 | Checksum on each outbound packet |
| Sequence numbers | Per-connection counter to detect packet loss/reorder |
| Opcode table | Binary opcode map identical to the original TFS |

---

## Contributing

We welcome community contributions! Please review our [Contributing Guidelines](CONTRIBUTING.md) for details on our code conventions, development workflow, and PR process.

Before opening a pull request, please make sure your changes pass the race detector and linters:

```bash
make test-race
make lint
```

---

## License

Licensed under the GNU General Public License v2.0 — see [LICENSE.md](LICENSE.md) for details.
