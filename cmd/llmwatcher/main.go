package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/faffige/llmwatcher/internal/config"
	"github.com/faffige/llmwatcher/internal/pipeline"
	"github.com/faffige/llmwatcher/internal/provider"
	"github.com/faffige/llmwatcher/internal/provider/openai"
	"github.com/faffige/llmwatcher/internal/proxy"
	"github.com/faffige/llmwatcher/internal/storage/sqlite"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "", "path to config file (YAML)")
	dbPath := flag.String("db", "llmwatcher.db", "path to SQLite database file")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("starting llmwatcher", "version", version)

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Storage.
	store, err := sqlite.New(*dbPath)
	if err != nil {
		logger.Error("failed to open database", "path", *dbPath, "error", err)
		os.Exit(1)
	}
	defer store.Close()
	logger.Info("database opened", "path", *dbPath)

	// Pipeline.
	pl := pipeline.New(store, 256, logger)
	defer pl.Close()

	parsers := map[string]provider.Parser{
		"openai": openai.New(),
	}

	proxyServer := proxy.New(cfg, parsers, pl.Submit, logger)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.ProxyPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      proxyServer.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("proxy listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	logger.Info("stopped")
}
