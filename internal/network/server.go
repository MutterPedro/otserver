package network

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

// Config holds configuration for the Server.
type Config struct {
	Address        string
	MaxConnections int
}

// Server is a TCP server that accepts connections up to a configured maximum.
type Server struct {
	cfg Config
}

// NewServer creates a new Server with the given configuration.
func NewServer(cfg Config) *Server {
	return &Server{cfg: cfg}
}

// ListenAndServe binds to the configured address, signals readiness on the ready
// channel, and accepts connections until ctx is cancelled. It returns nil on a
// clean context-driven shutdown.
func (s *Server) ListenAndServe(ctx context.Context, ready chan<- string) error {
	ln, err := net.Listen("tcp", s.cfg.Address)
	if err != nil {
		return err
	}

	var (
		wg      sync.WaitGroup
		count   int32
		maxConn = int32(s.cfg.MaxConnections)
	)

	// Close the listener when the context is cancelled so ln.Accept() unblocks.
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	ready <- ln.Addr().String()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// If the context was cancelled, treat as a clean shutdown.
			select {
			case <-ctx.Done():
				wg.Wait()
				return nil
			default:
				return err
			}
		}

		current := atomic.AddInt32(&count, 1)
		if maxConn > 0 && current > maxConn {
			atomic.AddInt32(&count, -1)
			_ = conn.Close()
			continue
		}

		wg.Add(1)
		go func(c net.Conn) {
			defer func() {
				_ = c.Close()
				atomic.AddInt32(&count, -1)
				wg.Done()
			}()

			// Drain the connection, but also watch for context cancellation so
			// that connections accepted just before shutdown are not missed.
			done := make(chan struct{})
			go func() {
				_, _ = io.Copy(io.Discard, c)
				close(done)
			}()

			select {
			case <-done:
				// Connection closed naturally.
			case <-ctx.Done():
				// Context cancelled: close the connection to unblock the drain
				// goroutine, then wait for it to finish.
				_ = c.Close()
				<-done
			}
		}(conn)
	}
}
