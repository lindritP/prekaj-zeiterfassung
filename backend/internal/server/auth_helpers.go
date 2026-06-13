package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

const refreshCookieName = "prekaj_refresh"

// normalizeEmail canonicalises an email for storage and lookup.
func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// serverError logs the underlying error (request-id aware) and writes a 500.
func (s *Server) serverError(w http.ResponseWriter, r *http.Request, err error) {
	platform.LoggerFrom(r.Context(), s.log).Error("internal error", "err", err)
	platform.WriteError(w, http.StatusInternalServerError, "internal_error", "Interner Serverfehler.")
}

// setRefreshCookie writes the rotating refresh token as an httpOnly cookie (web).
// Secure is enabled only in production so http://localhost works in dev.
func (s *Server) setRefreshCookie(w http.ResponseWriter, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    value,
		Path:     "/api/v1/auth",
		Expires:  expires,
		HttpOnly: true,
		Secure:   s.cfg.IsProd(),
		SameSite: http.SameSiteStrictMode,
	})
}

// clearRefreshCookie expires the refresh cookie (web logout).
func (s *Server) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cfg.IsProd(),
		SameSite: http.SameSiteStrictMode,
	})
}
