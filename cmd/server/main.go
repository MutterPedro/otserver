package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/MutterPedro/otserver/internal/config"
	"github.com/MutterPedro/otserver/internal/network"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "config.toml", "path to TOML config file")
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config %q: %w", *configPath, err)
	}

	srv := network.NewServer(network.Config{
		Address:        cfg.Server.Address,
		MaxConnections: cfg.Server.MaxConnections,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("shutdown signal received")
		cancel()
	}()

	ready := make(chan string, 1)
	go func() {
		addr := <-ready
		logger.Info("server listening", zap.String("address", addr))
	}()

	if err := srv.ListenAndServe(ctx, ready); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	logger.Info("server stopped cleanly")

	return nil
}

func loadConfig(path string) (config.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return config.Config{}, err
	}
	defer func() { _ = f.Close() }()

	return config.LoadFromReader(f)
}
