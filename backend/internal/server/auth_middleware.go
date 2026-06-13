package server

import (
	"net/http"
	"strings"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

// requireAuth validates the Bearer access token and injects the Identity.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || raw == "" {
			platform.WriteError(w, http.StatusUnauthorized, "unauthorized", "Kein gültiges Token.")
			return
		}
		ident, err := s.issuer.Verify(raw)
		if err != nil {
			platform.WriteError(w, http.StatusUnauthorized, "unauthorized", "Kein gültiges Token.")
			return
		}
		next.ServeHTTP(w, r.WithContext(withIdentity(r.Context(), ident)))
	})
}

// requireAdmin must be chained AFTER requireAuth.
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ident, ok := identityFrom(r.Context())
		if !ok || ident.Rolle != "admin" {
			platform.WriteError(w, http.StatusForbidden, "forbidden", "Nur für Administratoren.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
