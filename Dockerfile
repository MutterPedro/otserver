# ---- Build stage --------------------------------------------------------
FROM golang:1.26-bookworm AS builder

WORKDIR /src

# Download dependencies in a separate layer so they are cached independently
# of source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Produce a fully-static binary:
#   CGO_ENABLED=0  — no cgo, no libc dependency → compatible with distroless/static
#   -trimpath      — strip host file-system paths from the binary
#   -w -s          — omit DWARF debug info and the symbol table (smaller binary)
RUN CGO_ENABLED=0 GOOS=linux \
    go build \
      -trimpath \
      -ldflags="-w -s" \
      -o /out/server \
      ./cmd/server/

# ---- Runtime stage -------------------------------------------------------
# distroless/static contains only:
#   • CA certificates  (for TLS dial-outs)
#   • /etc/passwd      (non-root user support)
#   • tzdata           (time-zone data)
# Nothing else — no shell, no package manager, minimal attack surface.
FROM gcr.io/distroless/static-debian12

# Run as the built-in non-root user provided by distroless.
USER nonroot:nonroot

COPY --from=builder --chown=nonroot:nonroot /out/server /server

# Default game-server port. Override with -p at `docker run` time.
EXPOSE 7172

ENTRYPOINT ["/server"]

# Mount your config file at /etc/otserver/config.toml, e.g.:
#   docker run -v $(pwd)/config.toml:/etc/otserver/config.toml otserver
CMD ["--config", "/etc/otserver/config.toml"]
