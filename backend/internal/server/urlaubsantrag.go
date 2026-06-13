package server

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/db"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

const dateFormat = "2006-01-02"

// --- DTOs -------------------------------------------------------------------

type urlaubsantragResponse struct {
	ID             uuid.UUID  `json:"id"`
	ArbeiterID     uuid.UUID  `json:"arbeiter_id"`
	VonDatum       string     `json:"von_datum"`
	BisDatum       string     `json:"bis_datum"`
	Typ            string     `json:"typ"`
	Status         string     `json:"status"`
	Grund          string     `json:"grund"`
	EntschiedenVon *uuid.UUID `json:"entschieden_von"`
	EntschiedenAm  *string    `json:"entschieden_am"`
	CreatedAt      string     `json:"created_at"`
}

func toUrlaubResponse(u db.Urlaubsantrag) urlaubsantragResponse {
	resp := urlaubsantragResponse{
		ID:             u.ID,
		ArbeiterID:     u.ArbeiterID,
		VonDatum:       u.VonDatum.Format(dateFormat),
		BisDatum:       u.BisDatum.Format(dateFormat),
		Typ:            u.Typ,
		Status:         u.Status,
		Grund:          u.Grund,
		EntschiedenVon: u.EntschiedenVon,
		CreatedAt:      u.CreatedAt.UTC().Format(timeFormat),
	}
	if u.EntschiedenAm != nil {
		s := u.EntschiedenAm.UTC().Format(timeFormat)
		resp.EntschiedenAm = &s
	}
	return resp
}

type createUrlaubRequest struct {
	VonDatum string `json:"von_datum" validate:"required,datetime=2006-01-02"`
	BisDatum string `json:"bis_datum" validate:"required,datetime=2006-01-02"`
	Typ      string `json:"typ"       validate:"omitempty,oneof=urlaub krankheit sonstige"`
	Grund    string `json:"grund"     validate:"omitempty,max=1000"`
}

type decideUrlaubRequest struct {
	Status string `json:"status" validate:"required,oneof=genehmigt abgelehnt"`
}

// parseDatumRange reads optional ?von=&bis= date-only (YYYY-MM-DD) query params.
func parseDatumRange(w http.ResponseWriter, r *http.Request) (von, bis *time.Time, ok bool) {
	parse := func(name string) (*time.Time, bool) {
		raw := r.URL.Query().Get(name)
		if raw == "" {
			return nil, true
		}
		t, err := time.Parse(dateFormat, raw)
		if err != nil {
			platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültiges "+name+"-Datum (JJJJ-MM-TT).")
			return nil, false
		}
		return &t, true
	}
	von, ok = parse("von")
	if !ok {
		return nil, nil, false
	}
	bis, ok = parse("bis")
	if !ok {
		return nil, nil, false
	}
	return von, bis, true
}

// --- Worker handlers (requireAuth, scoped to identity.ArbeiterID) -----------

func (s *Server) handleCreateUrlaub(w http.ResponseWriter, r *http.Request) {
	ident, _ := identityFrom(r.Context())
	var req createUrlaubRequest
	if !s.decodeAndValidate(w, r, &req) {
		return
	}
	von, err := time.Parse(dateFormat, req.VonDatum)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültiges von_datum.")
		return
	}
	bis, err := time.Parse(dateFormat, req.BisDatum)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültiges bis_datum.")
		return
	}
	if bis.Before(von) {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "bis_datum darf nicht vor von_datum liegen.")
		return
	}
	typ := req.Typ
	if typ == "" {
		typ = "urlaub"
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	u, err := s.queries.CreateUrlaubsantrag(r.Context(), db.CreateUrlaubsantragParams{
		ID:         id,
		ArbeiterID: ident.ArbeiterID,
		VonDatum:   von,
		BisDatum:   bis,
		Typ:        typ,
		Grund:      req.Grund,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusCreated, toUrlaubResponse(u))
}

func (s *Server) handleListOwnUrlaub(w http.ResponseWriter, r *http.Request) {
	ident, _ := identityFrom(r.Context())
	rows, err := s.queries.ListOwnUrlaubsantrag(r.Context(), ident.ArbeiterID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	out := make([]urlaubsantragResponse, 0, len(rows))
	for _, u := range rows {
		out = append(out, toUrlaubResponse(u))
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (s *Server) handleDeleteUrlaub(w http.ResponseWriter, r *http.Request) {
	ident, _ := identityFrom(r.Context())
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	u, err := s.queries.GetUrlaubsantragByIDForArbeiter(r.Context(), db.GetUrlaubsantragByIDForArbeiterParams{
		ID:         id,
		ArbeiterID: ident.ArbeiterID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Antrag nicht gefunden.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if u.Status != "offen" {
		platform.WriteError(w, http.StatusConflict, "not_offen", "Nur offene Anträge können gelöscht werden.")
		return
	}
	if err := s.queries.DeleteUrlaubsantrag(r.Context(), db.DeleteUrlaubsantragParams{
		ID:         id,
		ArbeiterID: ident.ArbeiterID,
	}); err != nil {
		s.serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Admin handlers (requireAuth + requireAdmin) ----------------------------

func (s *Server) handleAdminListUrlaub(w http.ResponseWriter, r *http.Request) {
	var statusFilter *string
	if raw := r.URL.Query().Get("status"); raw != "" {
		if raw != "offen" && raw != "genehmigt" && raw != "abgelehnt" {
			platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültiger Status-Filter.")
			return
		}
		statusFilter = &raw
	}
	von, bis, ok := parseDatumRange(w, r)
	if !ok {
		return
	}
	rows, err := s.queries.AdminListUrlaubsantrag(r.Context(), db.AdminListUrlaubsantragParams{
		Status: statusFilter,
		Von:    von,
		Bis:    bis,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	out := make([]urlaubsantragResponse, 0, len(rows))
	for _, u := range rows {
		out = append(out, toUrlaubResponse(u))
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (s *Server) handleDecideUrlaub(w http.ResponseWriter, r *http.Request) {
	ident, _ := identityFrom(r.Context())
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req decideUrlaubRequest
	if !s.decodeAndValidate(w, r, &req) {
		return
	}
	cur, err := s.queries.GetUrlaubsantragByID(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Antrag nicht gefunden.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if cur.Status != "offen" {
		platform.WriteError(w, http.StatusConflict, "already_decided", "Antrag wurde bereits entschieden.")
		return
	}
	u, err := s.queries.DecideUrlaubsantrag(r.Context(), db.DecideUrlaubsantragParams{
		Status:         req.Status,
		EntschiedenVon: ident.ArbeiterID,
		ID:             id,
	})
	if errors.Is(err, pgx.ErrNoRows) { // race: decided between get and update
		platform.WriteError(w, http.StatusConflict, "already_decided", "Antrag wurde bereits entschieden.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toUrlaubResponse(u))
}
