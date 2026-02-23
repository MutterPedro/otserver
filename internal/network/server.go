// Package network implements the server-side network layer for the OTServer
// protocol. It handles TCP connections, packet framing, and connection lifecycle.
package network

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

type Config struct {
	Address        string
	MaxConnections int
}

type Server struct {
	cfg Config
}

func NewServer(cfg Config) *Server {
	return &Server{cfg: cfg}
}

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

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	ready <- ln.Addr().String()

	for {
		conn, err := ln.Accept()
		if err != nil {
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
			case <-ctx.Done():
				_ = c.Close()
				<-done
			}
		}(conn)
	}
}
