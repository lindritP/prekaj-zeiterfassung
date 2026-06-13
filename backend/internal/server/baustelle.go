package server

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/db"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

type baustelleResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Adresse   string    `json:"adresse"`
	Aktiv     bool      `json:"aktiv"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

func toBaustelleResponse(b db.Baustelle) baustelleResponse {
	return baustelleResponse{
		ID:        b.ID,
		Name:      b.Name,
		Adresse:   b.Adresse,
		Aktiv:     b.Aktiv,
		CreatedAt: b.CreatedAt.UTC().Format(timeFormat),
		UpdatedAt: b.UpdatedAt.UTC().Format(timeFormat),
	}
}

type createBaustelleRequest struct {
	Name    string `json:"name"    validate:"required,max=200"`
	Adresse string `json:"adresse" validate:"omitempty,max=500"`
}

type updateBaustelleRequest struct {
	Name    *string `json:"name"    validate:"omitempty,max=200"`
	Adresse *string `json:"adresse" validate:"omitempty,max=500"`
	Aktiv   *bool   `json:"aktiv"`
}

func (s *Server) handleListBaustellen(w http.ResponseWriter, r *http.Request) {
	aktiv, ok := parseAktivFilter(w, r)
	if !ok {
		return
	}
	rows, err := s.queries.ListBaustellen(r.Context(), aktiv)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	out := make([]baustelleResponse, 0, len(rows))
	for _, b := range rows {
		out = append(out, toBaustelleResponse(b))
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateBaustelle(w http.ResponseWriter, r *http.Request) {
	var req createBaustelleRequest
	if !s.decodeAndValidate(w, r, &req) {
		return
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	b, err := s.queries.CreateBaustelle(r.Context(), db.CreateBaustelleParams{
		ID:      id,
		Name:    req.Name,
		Adresse: req.Adresse,
		Aktiv:   true,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusCreated, toBaustelleResponse(b))
}

func (s *Server) handleGetBaustelle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	b, err := s.queries.GetBaustelleByID(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Baustelle nicht gefunden.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toBaustelleResponse(b))
}

func (s *Server) handlePatchBaustelle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req updateBaustelleRequest
	if !s.decodeAndValidate(w, r, &req) {
		return
	}
	b, err := s.queries.UpdateBaustelle(r.Context(), db.UpdateBaustelleParams{
		ID:      id,
		Name:    req.Name,
		Adresse: req.Adresse,
		Aktiv:   req.Aktiv,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Baustelle nicht gefunden.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toBaustelleResponse(b))
}

func (s *Server) handleDeactivateBaustelle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	b, err := s.queries.DeactivateBaustelle(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Baustelle nicht gefunden.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toBaustelleResponse(b))
}
