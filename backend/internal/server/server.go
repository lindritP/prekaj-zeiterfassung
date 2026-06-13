// Package server wires the HTTP router, middleware and handlers.
package server

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/config"
)

// Server holds shared dependencies for HTTP handlers.
type Server struct {
	cfg  config.Config
	log  *slog.Logger
	pool *pgxpool.Pool
}

// New builds the fully wired http.Handler (router + middleware + routes).
func New(cfg config.Config, log *slog.Logger, pool *pgxpool.Pool) http.Handler {
	s := &Server{cfg: cfg, log: log, pool: pool}
	return s.routes()
}
