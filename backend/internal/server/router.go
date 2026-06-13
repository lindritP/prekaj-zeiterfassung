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

		// Zeiterfassung — Arbeiter (Phase 4): requireAuth, Queries auf identity.ArbeiterID gescoped.
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Route("/zeit", func(r chi.Router) {
				r.Post("/start", s.handleStartZeit)
				r.Post("/stop", s.handleStopZeit)
				r.Get("/laufend", s.handleGetLaufend) // statisch VOR /{id}
				r.Get("/", s.handleListOwnZeit)
				r.Patch("/{id}", s.handlePatchZeit)
			})

			// Urlaubsanträge — Arbeiter (Phase 5).
			r.Route("/urlaub", func(r chi.Router) {
				r.Post("/", s.handleCreateUrlaub)
				r.Get("/", s.handleListOwnUrlaub)
				r.Delete("/{id}", s.handleDeleteUrlaub)
			})

			// Überstunden — Arbeiter (Phase 6): eigener Monats-Saldo.
			r.Get("/ueberstunden", s.handleOwnUeberstunden)

			// Dokumente — Arbeiter (Phase 7): eigene Liste + Download (own-or-admin im Handler).
			r.Get("/dokumente", s.handleListOwnDokument)
			r.Get("/dokumente/{id}/download", s.handleDownloadDokument)
		})

		// Admin-Stammdaten (Phase 3): alles hinter requireAuth + requireAdmin.
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth, s.requireAdmin)

			r.Route("/arbeiter", func(r chi.Router) {
				r.Get("/", s.handleListArbeiter)
				r.Post("/", s.handleCreateArbeiter)
				r.Get("/{id}", s.handleGetArbeiter)
				r.Patch("/{id}", s.handlePatchArbeiter)
				r.Delete("/{id}", s.handleDeactivateArbeiter)
			})

			r.Route("/baustellen", func(r chi.Router) {
				r.Get("/", s.handleListBaustellen)
				r.Post("/", s.handleCreateBaustelle)
				r.Get("/{id}", s.handleGetBaustelle)
				r.Patch("/{id}", s.handlePatchBaustelle)
				r.Delete("/{id}", s.handleDeactivateBaustelle)
			})

			// Zeiterfassung — Admin (Phase 4): alle Buchungen, Filter + Summen.
			r.Get("/admin/zeit", s.handleAdminListZeit)

			// Urlaubsanträge — Admin (Phase 5): Liste (Filter) + Entscheidung.
			r.Route("/admin/urlaub", func(r chi.Router) {
				r.Get("/", s.handleAdminListUrlaub)
				r.Patch("/{id}", s.handleDecideUrlaub)
			})

			// Überstunden — Admin (Phase 6): Saldo aller (oder ?arbeiter=) je Monat.
			r.Get("/admin/ueberstunden", s.handleAdminUeberstunden)

			// PDF & Dokumente — Admin (Phase 7).
			r.Get("/admin/berichte/monat", s.handleMonatsbericht)
			r.Post("/admin/dokumente", s.handleUploadDokument)
			r.Get("/admin/dokumente", s.handleAdminListDokument)
		})
	})

	return r
}
