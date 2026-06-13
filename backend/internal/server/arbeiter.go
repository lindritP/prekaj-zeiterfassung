package server

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/db"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

// arbeiterResponse is the admin-facing view. It deliberately OMITS passwort_hash.
type arbeiterResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	Rolle         string    `json:"rolle"`
	Wochenstunden string    `json:"wochenstunden"`
	Stundenlohn   string    `json:"stundenlohn"`
	Aktiv         bool      `json:"aktiv"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
}

func toArbeiterResponse(a db.Arbeiter) arbeiterResponse {
	return arbeiterResponse{
		ID:            a.ID,
		Name:          a.Name,
		Email:         a.Email,
		Rolle:         a.Rolle,
		Wochenstunden: a.Wochenstunden,
		Stundenlohn:   a.Stundenlohn,
		Aktiv:         a.Aktiv,
		CreatedAt:     a.CreatedAt.UTC().Format(timeFormat),
		UpdatedAt:     a.UpdatedAt.UTC().Format(timeFormat),
	}
}

type createArbeiterRequest struct {
	Name          string `json:"name"          validate:"required,max=200"`
	Email         string `json:"email"         validate:"required,email,max=255"`
	Passwort      string `json:"passwort"      validate:"required,min=8,max=72"`
	Rolle         string `json:"rolle"         validate:"omitempty,oneof=arbeiter admin"`
	Wochenstunden string `json:"wochenstunden" validate:"omitempty,numeric"`
	Stundenlohn   string `json:"stundenlohn"   validate:"omitempty,numeric"`
}

// updateArbeiterRequest uses pointers so "absent" (nil) differs from "set to empty".
type updateArbeiterRequest struct {
	Name          *string `json:"name"          validate:"omitempty,max=200"`
	Email         *string `json:"email"         validate:"omitempty,email,max=255"`
	Passwort      *string `json:"passwort"      validate:"omitempty,min=8,max=72"`
	Rolle         *string `json:"rolle"         validate:"omitempty,oneof=arbeiter admin"`
	Wochenstunden *string `json:"wochenstunden" validate:"omitempty,numeric"`
	Stundenlohn   *string `json:"stundenlohn"   validate:"omitempty,numeric"`
	Aktiv         *bool   `json:"aktiv"`
}

func (s *Server) handleListArbeiter(w http.ResponseWriter, r *http.Request) {
	aktiv, ok := parseAktivFilter(w, r)
	if !ok {
		return
	}
	rows, err := s.queries.ListArbeiter(r.Context(), aktiv)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	out := make([]arbeiterResponse, 0, len(rows))
	for _, a := range rows {
		out = append(out, toArbeiterResponse(a))
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateArbeiter(w http.ResponseWriter, r *http.Request) {
	var req createArbeiterRequest
	if !s.decodeAndValidate(w, r, &req) {
		return
	}
	hash, err := s.hasher.Hash(req.Passwort)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	rolle := req.Rolle
	if rolle == "" {
		rolle = "arbeiter"
	}
	a, err := s.queries.CreateArbeiter(r.Context(), db.CreateArbeiterParams{
		ID:            id,
		Name:          req.Name,
		Email:         normalizeEmail(req.Email),
		PasswortHash:  hash,
		Rolle:         rolle,
		Wochenstunden: defaultNumeric(req.Wochenstunden),
		Stundenlohn:   defaultNumeric(req.Stundenlohn),
		Aktiv:         true,
	})
	if isUniqueViolation(err) {
		platform.WriteError(w, http.StatusConflict, "email_taken", "E-Mail ist bereits vergeben.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusCreated, toArbeiterResponse(a))
}

func (s *Server) handleGetArbeiter(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	a, err := s.queries.GetArbeiterByID(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Arbeiter nicht gefunden.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toArbeiterResponse(a))
}

func (s *Server) handlePatchArbeiter(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req updateArbeiterRequest
	if !s.decodeAndValidate(w, r, &req) {
		return
	}

	params := db.UpdateArbeiterParams{
		ID:            id,
		Name:          req.Name,
		Rolle:         req.Rolle,
		Wochenstunden: req.Wochenstunden,
		Stundenlohn:   req.Stundenlohn,
		Aktiv:         req.Aktiv,
	}
	if req.Email != nil {
		norm := normalizeEmail(*req.Email)
		params.Email = &norm
	}
	if req.Passwort != nil {
		hash, err := s.hasher.Hash(*req.Passwort)
		if err != nil {
			s.serverError(w, r, err)
			return
		}
		params.PasswortHash = &hash
	}

	a, err := s.queries.UpdateArbeiter(r.Context(), params)
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Arbeiter nicht gefunden.")
		return
	}
	if isUniqueViolation(err) {
		platform.WriteError(w, http.StatusConflict, "email_taken", "E-Mail ist bereits vergeben.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toArbeiterResponse(a))
}

func (s *Server) handleDeactivateArbeiter(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	a, err := s.queries.DeactivateArbeiter(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Arbeiter nicht gefunden.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toArbeiterResponse(a))
}
