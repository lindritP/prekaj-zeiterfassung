// Package db owns the PostgreSQL connection pool (pgx v5 pgxpool).
// sqlc-generated code (db.go, models.go, *.sql.go) also lands in this package.
package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxuuid "github.com/vgarvardt/pgx-google-uuid/v5"
)

// NewPool builds a pgx connection pool from a DATABASE_URL and verifies it with a
// ping. (Named NewPool to avoid clashing with sqlc's generated New(DBTX) *Queries.)
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	// Conservative pool defaults for local dev; tune per environment later.
	cfg.MaxConns = 10
	cfg.MaxConnIdleTime = 5 * time.Minute

	// Register the google/uuid codec on each connection so native `uuid` columns
	// encode/decode as uuid.UUID over the binary protocol.
	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// Ping verifies DB connectivity; used by the /readyz endpoint.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return pool.Ping(pingCtx)
}
