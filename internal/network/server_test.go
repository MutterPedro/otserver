package network_test

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/MutterPedro/otserver/internal/network"
)

// startTestServer is a helper that starts a Server with the given Config and
// returns the bound address. It registers a cleanup that cancels the context
// and waits for the server to exit.
func startTestServer(t *testing.T, cfg network.Config) string {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())

	srv := network.NewServer(cfg)
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		done <- srv.ListenAndServe(ctx, ready)
	}()

	var addr string
	select {
	case addr = <-ready:
	case <-time.After(3 * time.Second):
		cancel()
		t.Fatal("server did not become ready within 3s")
	}

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("server shutdown error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("server did not shut down within 5s")
		}
	})

	return addr
}

// TestAcceptance_ServerStartsAndAcceptsConnections verifies that the TCP server:
//   - binds to an ephemeral port
//   - accepts an inbound TCP connection
//   - shuts down gracefully when the context is cancelled
func TestAcceptance_ServerStartsAndAcceptsConnections(t *testing.T) {
	t.Parallel()

	addr := startTestServer(t, network.Config{Address: "127.0.0.1:0", MaxConnections: 10})

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("expected to connect to server at %s, got error: %v", addr, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
}

// TestGracefulShutdown_GoroutineLeaks verifies that all server-owned goroutines
// are released after context cancellation, leaving no leaked goroutines.
func TestGracefulShutdown_GoroutineLeaks(t *testing.T) {
	// Not parallel: compares global goroutine count.

	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())

	srv := network.NewServer(network.Config{Address: "127.0.0.1:0", MaxConnections: 10})
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		done <- srv.ListenAndServe(ctx, ready)
	}()

	addr := <-ready

	var conns []net.Conn
	for range 5 {
		c, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		conns = append(conns, c)
	}

	runtime.Gosched()
	time.Sleep(50 * time.Millisecond)

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("server exit error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down within 5s")
	}

	for _, c := range conns {
		_ = c.Close()
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= baseline+1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	leaked := runtime.NumGoroutine() - baseline
	t.Errorf("goroutine leak: %d goroutine(s) above baseline after shutdown", leaked)
}

// TestServer_MaxConnectionsLimit verifies that the server rejects connections
// that exceed MaxConnections by closing them immediately.
func TestServer_MaxConnectionsLimit(t *testing.T) {
	t.Parallel()

	const limit = 2

	addr := startTestServer(t, network.Config{Address: "127.0.0.1:0", MaxConnections: limit})

	allowed := make([]net.Conn, limit)
	for i := range limit {
		c, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			t.Fatalf("connection %d (within limit): unexpected dial error: %v", i+1, err)
		}
		allowed[i] = c
	}
	defer func() {
		for _, c := range allowed {
			_ = c.Close()
		}
	}()

	// The (limit+1)th connection should be rejected: the server closes it so any
	// read on the client side must return io.EOF or an error promptly.
	excess, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		// Some implementations refuse the connection outright at TCP level.
		return
	}
	defer func() { _ = excess.Close() }()

	if err := excess.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	buf := make([]byte, 1)
	_, err = excess.Read(buf)
	if err == nil {
		t.Fatal("expected excess connection to be closed by server, but Read succeeded")
	}
}

// TestServer_ConcurrentConnections verifies that the server handles many
// simultaneous connections without data races.
func TestServer_ConcurrentConnections(t *testing.T) {
	t.Parallel()

	const numClients = 100

	addr := startTestServer(t, network.Config{Address: "127.0.0.1:0", MaxConnections: numClients})

	var wg sync.WaitGroup
	errors := make(chan error, numClients)

	for i := range numClients {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
			if err != nil {
				errors <- fmt.Errorf("client %d: dial failed: %w", id, err)
				return
			}
			defer func() { _ = conn.Close() }()
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}
