// Command api is the HTTP entrypoint for the Prekaj-Zeiterfassung backend.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/config"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/db"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/server"
)

func main() {
	if err := run(); err != nil {
		// Logger may not exist yet; use a minimal fallback.
		platform.NewLogger("error", false).Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Config (loads .env locally, then parses env vars).
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// 2. Logger (JSON in prod, text locally).
	log := platform.NewLogger(cfg.LogLevel, cfg.IsProd())
	log.Info("starting api", "env", cfg.Env, "port", cfg.Port)

	// 3. Root context cancelled on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. DB pool + startup ping.
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Info("database connected")

	// 5. Router + HTTP server.
	handler := server.New(cfg, log, pool)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 6. Serve in the background; report listen errors over a channel.
	serverErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// 7. Block until a signal or a server error.
	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	// 8. Graceful shutdown with a bounded, fresh timeout (not the cancelled root ctx).
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
		return err
	}
	log.Info("shutdown complete")
	return nil
}
