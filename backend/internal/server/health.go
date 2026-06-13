package server

import (
	"net/http"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/db"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

// handleHealthz is the liveness probe: always 200 if the process is up.
func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReadyz is the readiness probe: 200 only if the DB ping succeeds, else 503.
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(r.Context(), s.pool); err != nil {
		platform.WriteError(w, http.StatusServiceUnavailable, "not_ready", "Datenbank nicht erreichbar")
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
