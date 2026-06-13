package server

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

// timeFormat is the wire format for timestamps (RFC3339 in UTC).
const timeFormat = "2006-01-02T15:04:05Z07:00"

// parseIDParam reads {id} from the path and parses it as a UUID (400 on bad input).
// A non-UUID id can never match a row, so we reject it early.
func parseIDParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültige ID.")
		return uuid.Nil, false
	}
	return id, true
}

// parseAktivFilter reads the optional ?aktiv=true|false query param.
// Absent => nil (no filter). Invalid value => 400.
func parseAktivFilter(w http.ResponseWriter, r *http.Request) (*bool, bool) {
	switch r.URL.Query().Get("aktiv") {
	case "":
		return nil, true
	case "true":
		v := true
		return &v, true
	case "false":
		v := false
		return &v, true
	default:
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültiger aktiv-Filter.")
		return nil, false
	}
}

// isUniqueViolation reports whether err is a Postgres unique_violation (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// defaultNumeric maps an empty optional numeric string to "0" so create always
// receives a valid numeric literal (matches the column default).
func defaultNumeric(s string) string {
	if s == "" {
		return "0"
	}
	return s
}
