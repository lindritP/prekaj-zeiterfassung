// Package server wires the HTTP router, middleware and handlers.
package server

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/auth"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/config"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/db"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/storage"
)

// Server holds shared dependencies for HTTP handlers.
type Server struct {
	cfg       config.Config
	log       *slog.Logger
	pool      *pgxpool.Pool
	queries   *db.Queries
	hasher    auth.Hasher
	issuer    *auth.TokenIssuer
	validate  *validator.Validate
	store     storage.Storage
	dummyHash string // bcrypt hash for timing-safe login (anti-enumeration)
}

// New builds the fully wired http.Handler (router + middleware + routes).
// Returns an error if the auth issuer can't be constructed (e.g. weak JWT secret).
func New(cfg config.Config, log *slog.Logger, pool *pgxpool.Pool) (http.Handler, error) {
	issuer, err := auth.NewTokenIssuer(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		return nil, err
	}
	hasher := auth.NewHasher(cfg.BcryptCost)

	// Precompute a dummy bcrypt hash so an unknown-email login still spends a
	// bcrypt compare — keeps login timing constant (no user enumeration).
	seed := make([]byte, 16)
	if _, err := rand.Read(seed); err != nil {
		return nil, err
	}
	dummyHash, err := hasher.Hash(hex.EncodeToString(seed))
	if err != nil {
		return nil, err
	}

	// Validator: report errors using the json field name (e.g. "wochenstunden").
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	store, err := storage.NewLocal(cfg.DokumenteDir)
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:       cfg,
		log:       log,
		pool:      pool,
		queries:   db.New(pool),
		hasher:    hasher,
		issuer:    issuer,
		validate:  v,
		store:     store,
		dummyHash: dummyHash,
	}
	return s.routes(), nil
}
