// Command seed creates or updates the initial admin (the owner) idempotently.
// Reads SEED_ADMIN_* from the environment (.env locally). Secrets stay out of migrations.
package main

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/auth"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/config"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/db"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

func main() {
	if err := run(); err != nil {
		platform.NewLogger("error", false).Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.SeedAdminEmail == "" || cfg.SeedAdminPassword == "" {
		return errors.New("SEED_ADMIN_EMAIL und SEED_ADMIN_PASSWORD müssen gesetzt sein")
	}
	log := platform.NewLogger(cfg.LogLevel, cfg.IsProd())

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	hash, err := auth.NewHasher(cfg.BcryptCost).Hash(cfg.SeedAdminPassword)
	if err != nil {
		return err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	a, err := db.New(pool).UpsertAdmin(ctx, db.UpsertAdminParams{
		ID:           id,
		Name:         cfg.SeedAdminName,
		Email:        strings.ToLower(strings.TrimSpace(cfg.SeedAdminEmail)),
		PasswortHash: hash,
	})
	if err != nil {
		return err
	}
	log.Info("admin seeded", "email", a.Email, "rolle", a.Rolle)
	return nil
}
