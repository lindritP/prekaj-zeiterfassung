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

// statutoryPause returns the legally-mandated break (minutes) for a worked span,
// per Austrian Arbeitszeitgesetz §11: a span over 6h (360 min) requires at least a
// 30-min break. Single source of truth — extend here for collective-agreement tiers.
func statutoryPause(spanMin int32) int32 {
	if spanMin > 360 {
		return 30
	}
	return 0
}

// dauerMinuten = (end - start) truncated to whole minutes, minus pause; clamped >= 0.
// No rounding to 5/15 (CLAUDE.md §13 default).
func dauerMinuten(start, end time.Time, pause int32) int32 {
	d := int32(end.Sub(start).Minutes()) - pause
	if d < 0 {
		return 0
	}
	return d
}

// validateSpan enforces end>start and pause<=span. On violation it writes a 400 and
// returns false. Over-midnight is a non-issue: UTC timestamps make a booking across
// midnight just end>start.
func validateSpan(w http.ResponseWriter, start, end time.Time, pause int32) bool {
	if !end.After(start) {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "end_zeit muss nach start_zeit liegen.")
		return false
	}
	if pause > int32(end.Sub(start).Minutes()) {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Pause überschreitet die Arbeitszeit.")
		return false
	}
	return true
}

func normTime(p *time.Time) *time.Time {
	if p == nil {
		return nil
	}
	u := p.UTC()
	return &u
}

// --- DTOs -------------------------------------------------------------------

type zeitbuchungResponse struct {
	ID           uuid.UUID  `json:"id"`
	ArbeiterID   uuid.UUID  `json:"arbeiter_id"`
	BaustelleID  *uuid.UUID `json:"baustelle_id"`
	StartZeit    string     `json:"start_zeit"`
	EndZeit      *string    `json:"end_zeit"`
	PauseMinuten int32      `json:"pause_minuten"`
	DauerMinuten *int32     `json:"dauer_minuten"`
	Notiz        string     `json:"notiz"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
}

func toZeitbuchungResponse(z db.Zeitbuchung) zeitbuchungResponse {
	resp := zeitbuchungResponse{
		ID:           z.ID,
		ArbeiterID:   z.ArbeiterID,
		BaustelleID:  z.BaustelleID,
		StartZeit:    z.StartZeit.UTC().Format(timeFormat),
		PauseMinuten: z.PauseMinuten,
		Notiz:        z.Notiz,
		CreatedAt:    z.CreatedAt.UTC().Format(timeFormat),
		UpdatedAt:    z.UpdatedAt.UTC().Format(timeFormat),
	}
	if z.EndZeit != nil {
		end := z.EndZeit.UTC().Format(timeFormat)
		resp.EndZeit = &end
		d := dauerMinuten(z.StartZeit, *z.EndZeit, z.PauseMinuten)
		resp.DauerMinuten = &d
	}
	return resp
}

type startZeitRequest struct {
	BaustelleID *uuid.UUID `json:"baustelle_id"`
	StartZeit   *time.Time `json:"start_zeit"` // RFC3339; absent => server now()
	Notiz       string     `json:"notiz" validate:"omitempty,max=1000"`
}

type stopZeitRequest struct {
	EndZeit *time.Time `json:"end_zeit"` // RFC3339; absent => server now()
}

type updateZeitRequest struct {
	BaustelleID  *uuid.UUID `json:"baustelle_id"`
	StartZeit    *time.Time `json:"start_zeit"`
	EndZeit      *time.Time `json:"end_zeit"`
	PauseMinuten *int32     `json:"pause_minuten" validate:"omitempty,min=0"` // explicit override
	Notiz        *string    `json:"notiz" validate:"omitempty,max=1000"`
}

type adminZeitArbeiterSumme struct {
	ArbeiterID   uuid.UUID `json:"arbeiter_id"`
	SummeMinuten int64     `json:"summe_minuten"`
	Anzahl       int64     `json:"anzahl"`
}

type adminZeitListResponse struct {
	Buchungen    []zeitbuchungResponse    `json:"buchungen"`
	SummeMinuten int64                    `json:"summe_minuten"`
	Anzahl       int64                    `json:"anzahl"`
	ProArbeiter  []adminZeitArbeiterSumme `json:"pro_arbeiter"`
}

// --- Worker handlers (requireAuth, scoped to identity.ArbeiterID) -----------

func (s *Server) handleStartZeit(w http.ResponseWriter, r *http.Request) {
	ident, _ := identityFrom(r.Context())
	var req startZeitRequest
	if !s.decodeAndValidate(w, r, &req) {
		return
	}

	start := time.Now().UTC()
	if req.StartZeit != nil {
		start = req.StartZeit.UTC()
		if start.After(time.Now().UTC()) {
			platform.WriteError(w, http.StatusBadRequest, "bad_request", "start_zeit darf nicht in der Zukunft liegen.")
			return
		}
	}

	id, err := uuid.NewV7()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	z, err := s.queries.StartZeitbuchung(r.Context(), db.StartZeitbuchungParams{
		ID:          id,
		ArbeiterID:  ident.ArbeiterID,
		BaustelleID: req.BaustelleID,
		StartZeit:   start,
		Notiz:       req.Notiz,
	})
	if isUniqueViolation(err) {
		platform.WriteError(w, http.StatusConflict, "running_exists", "Es läuft bereits eine Zeitbuchung.")
		return
	}
	if isForeignKeyViolation(err) {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Unbekannte Baustelle.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusCreated, toZeitbuchungResponse(z))
}

func (s *Server) handleStopZeit(w http.ResponseWriter, r *http.Request) {
	ident, _ := identityFrom(r.Context())
	var req stopZeitRequest
	if !s.decodeAndValidate(w, r, &req) {
		return
	}

	running, err := s.queries.GetRunningForArbeiter(r.Context(), ident.ArbeiterID)
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusConflict, "no_running", "Keine laufende Buchung.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	end := time.Now().UTC()
	if req.EndZeit != nil {
		end = req.EndZeit.UTC()
	}
	if !end.After(running.StartZeit) {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "end_zeit muss nach start_zeit liegen.")
		return
	}
	pause := statutoryPause(int32(end.Sub(running.StartZeit).Minutes()))

	z, err := s.queries.StopZeitbuchung(r.Context(), db.StopZeitbuchungParams{
		EndZeit:      end,
		PauseMinuten: pause,
		ArbeiterID:   ident.ArbeiterID,
	})
	if errors.Is(err, pgx.ErrNoRows) { // lost the race: another stop won
		platform.WriteError(w, http.StatusConflict, "no_running", "Keine laufende Buchung.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toZeitbuchungResponse(z))
}

func (s *Server) handleGetLaufend(w http.ResponseWriter, r *http.Request) {
	ident, _ := identityFrom(r.Context())
	z, err := s.queries.GetRunningForArbeiter(r.Context(), ident.ArbeiterID)
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteJSON(w, http.StatusOK, nil) // nichts läuft
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toZeitbuchungResponse(z))
}

func (s *Server) handleListOwnZeit(w http.ResponseWriter, r *http.Request) {
	ident, _ := identityFrom(r.Context())
	von, bis, ok := parseZeitraum(w, r)
	if !ok {
		return
	}
	rows, err := s.queries.ListOwnZeitbuchung(r.Context(), db.ListOwnZeitbuchungParams{
		ArbeiterID: ident.ArbeiterID,
		Von:        von,
		Bis:        bis,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	out := make([]zeitbuchungResponse, 0, len(rows))
	for _, z := range rows {
		out = append(out, toZeitbuchungResponse(z))
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (s *Server) handlePatchZeit(w http.ResponseWriter, r *http.Request) {
	ident, _ := identityFrom(r.Context())
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req updateZeitRequest
	if !s.decodeAndValidate(w, r, &req) {
		return
	}

	cur, err := s.queries.GetZeitbuchungByIDForArbeiter(r.Context(), db.GetZeitbuchungByIDForArbeiterParams{
		ID:         id,
		ArbeiterID: ident.ArbeiterID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Zeitbuchung nicht gefunden.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	// Effective start/end after the patch.
	start := cur.StartZeit
	if req.StartZeit != nil {
		start = req.StartZeit.UTC()
	}
	end := cur.EndZeit
	if req.EndZeit != nil {
		e := req.EndZeit.UTC()
		end = &e
	}

	// Resolve pause: explicit override wins; else recompute statutory if the span
	// changed on a completed booking; else leave unchanged.
	var pausePtr *int32
	switch {
	case req.PauseMinuten != nil:
		if end != nil && !validateSpan(w, start, *end, *req.PauseMinuten) {
			return
		}
		pausePtr = req.PauseMinuten
	case end != nil && (req.StartZeit != nil || req.EndZeit != nil):
		if !end.After(start) {
			platform.WriteError(w, http.StatusBadRequest, "bad_request", "end_zeit muss nach start_zeit liegen.")
			return
		}
		p := statutoryPause(int32(end.Sub(start).Minutes()))
		pausePtr = &p
	}

	z, err := s.queries.UpdateZeitbuchung(r.Context(), db.UpdateZeitbuchungParams{
		ID:           id,
		ArbeiterID:   ident.ArbeiterID,
		StartZeit:    normTime(req.StartZeit),
		EndZeit:      normTime(req.EndZeit),
		BaustelleID:  req.BaustelleID,
		PauseMinuten: pausePtr,
		Notiz:        req.Notiz,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Zeitbuchung nicht gefunden.")
		return
	}
	if isForeignKeyViolation(err) {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Unbekannte Baustelle.")
		return
	}
	if isUniqueViolation(err) {
		platform.WriteError(w, http.StatusConflict, "running_exists", "Es läuft bereits eine Zeitbuchung.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toZeitbuchungResponse(z))
}

// --- Admin handler (requireAuth + requireAdmin) -----------------------------

func (s *Server) handleAdminListZeit(w http.ResponseWriter, r *http.Request) {
	arbeiterID, ok := parseOptionalUUIDQuery(w, r, "arbeiter")
	if !ok {
		return
	}
	baustelleID, ok := parseOptionalUUIDQuery(w, r, "baustelle")
	if !ok {
		return
	}
	von, bis, ok := parseZeitraum(w, r)
	if !ok {
		return
	}

	rows, err := s.queries.AdminListZeitbuchung(r.Context(), db.AdminListZeitbuchungParams{
		ArbeiterID:  arbeiterID,
		BaustelleID: baustelleID,
		Von:         von,
		Bis:         bis,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	sum, err := s.queries.AdminSumZeitbuchung(r.Context(), db.AdminSumZeitbuchungParams{
		ArbeiterID:  arbeiterID,
		BaustelleID: baustelleID,
		Von:         von,
		Bis:         bis,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	perArb, err := s.queries.AdminSumZeitbuchungPerArbeiter(r.Context(), db.AdminSumZeitbuchungPerArbeiterParams{
		ArbeiterID:  arbeiterID,
		BaustelleID: baustelleID,
		Von:         von,
		Bis:         bis,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	buchungen := make([]zeitbuchungResponse, 0, len(rows))
	for _, z := range rows {
		buchungen = append(buchungen, toZeitbuchungResponse(z))
	}
	pro := make([]adminZeitArbeiterSumme, 0, len(perArb))
	for _, p := range perArb {
		pro = append(pro, adminZeitArbeiterSumme{ArbeiterID: p.ArbeiterID, SummeMinuten: p.SummeMinuten, Anzahl: p.Anzahl})
	}
	platform.WriteJSON(w, http.StatusOK, adminZeitListResponse{
		Buchungen:    buchungen,
		SummeMinuten: sum.SummeMinuten,
		Anzahl:       sum.Anzahl,
		ProArbeiter:  pro,
	})
}
