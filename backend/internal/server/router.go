package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

// routes builds the chi router with the standard middleware stack and mounts all
// business routes under /api/v1. Health endpoints live at the root so probes hit
// them directly. Auth middleware is added in Phase 2.
func (s *Server) routes() http.Handler {
	r := chi.NewRouter()

	// Middleware stack (CLAUDE.md §7): RequestID -> Logger -> Recoverer -> CORS -> RateLimit.
	r.Use(chimw.RequestID)
	r.Use(s.requestLogger) // structured slog request logger (middleware.go)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.cfg.CORSOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true, // Phase 2: refresh-token cookie (web)
		MaxAge:           300,
	}))
	r.Use(httprate.Limit(
		120, time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(func(w http.ResponseWriter, _ *http.Request) {
			platform.WriteError(w, http.StatusTooManyRequests, "rate_limited",
				"Zu viele Anfragen. Bitte später erneut versuchen.")
		}),
	))

	// Health endpoints — at root, NOT under /api/v1.
	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)

	// Versioned API surface.
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", s.handleLogin)
			r.Post("/refresh", s.handleRefresh)
			r.Post("/logout", s.handleLogout)
			r.With(s.requireAuth).Get("/me", s.handleMe)
		})

		// TEMPORÄRE DoD-Probe für Phase 2 — entfällt, sobald Phase 3 echte
		// Admin-Routen bringt.
		r.With(s.requireAuth, s.requireAdmin).Get("/admin/ping", func(w http.ResponseWriter, _ *http.Request) {
			platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "admin-ok"})
		})
	})

	return r
}
