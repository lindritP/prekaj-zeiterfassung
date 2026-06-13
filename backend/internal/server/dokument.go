package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/db"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

type dokumentResponse struct {
	ID         uuid.UUID `json:"id"`
	ArbeiterID uuid.UUID `json:"arbeiter_id"`
	Typ        string    `json:"typ"`
	Jahr       int32     `json:"jahr"`
	Monat      int32     `json:"monat"`
	Dateiname  string    `json:"dateiname"`
	MimeType   string    `json:"mime_type"`
	Groesse    int64     `json:"groesse"`
	CreatedAt  string    `json:"created_at"`
}

func toDokumentResponse(d db.Dokument) dokumentResponse {
	return dokumentResponse{
		ID:         d.ID,
		ArbeiterID: d.ArbeiterID,
		Typ:        d.Typ,
		Jahr:       d.Jahr,
		Monat:      d.Monat,
		Dateiname:  d.Dateiname,
		MimeType:   d.MimeType,
		Groesse:    d.Groesse,
		CreatedAt:  d.CreatedAt.UTC().Format(timeFormat),
	}
}

// handleUploadDokument (admin) stores an uploaded PDF as a Lohnzettel for an arbeiter.
func (s *Server) handleUploadDokument(w http.ResponseWriter, r *http.Request) {
	maxBytes := int64(s.cfg.MaxUploadMB) * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+1024) // + form overhead
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Datei zu groß oder ungültiges Formular.")
		return
	}

	arbeiterID, err := uuid.Parse(r.FormValue("arbeiter_id"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültige arbeiter_id.")
		return
	}
	typ := r.FormValue("typ")
	if typ == "" {
		typ = "lohnzettel"
	}
	if typ != "lohnzettel" && typ != "sonstige" {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültiger typ.")
		return
	}
	jahr, err := strconv.Atoi(r.FormValue("jahr"))
	if err != nil || jahr < 2000 || jahr > 2100 {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültiges jahr.")
		return
	}
	monat, err := strconv.Atoi(r.FormValue("monat"))
	if err != nil || monat < 1 || monat > 12 {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültiger monat (1-12).")
		return
	}

	file, header, err := r.FormFile("datei")
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Datei fehlt (Feld 'datei').")
		return
	}
	defer func() { _ = file.Close() }()
	buf, err := io.ReadAll(file)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Datei zu groß.")
		return
	}
	if len(buf) == 0 || !bytes.HasPrefix(buf, []byte("%PDF-")) {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Nur PDF-Dateien erlaubt.")
		return
	}

	// Arbeiter muss existieren (vermeidet verwaiste Dateien).
	if _, err := s.queries.GetArbeiterByID(r.Context(), arbeiterID); errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Arbeiter nicht gefunden.")
		return
	} else if err != nil {
		s.serverError(w, r, err)
		return
	}

	keyID, err := uuid.NewV7()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	key := keyID.String() + ".pdf"
	size, err := s.store.Save(r.Context(), key, bytes.NewReader(buf))
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	dok, err := s.queries.CreateDokument(r.Context(), db.CreateDokumentParams{
		ID:         id,
		ArbeiterID: arbeiterID,
		Typ:        typ,
		Jahr:       int32(jahr),
		Monat:      int32(monat),
		Dateiname:  filepath.Base(header.Filename),
		StorageKey: key,
		MimeType:   "application/pdf",
		Groesse:    size,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusCreated, toDokumentResponse(dok))
}

func (s *Server) handleListOwnDokument(w http.ResponseWriter, r *http.Request) {
	ident, _ := identityFrom(r.Context())
	rows, err := s.queries.ListOwnDokument(r.Context(), ident.ArbeiterID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	out := make([]dokumentResponse, 0, len(rows))
	for _, d := range rows {
		out = append(out, toDokumentResponse(d))
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (s *Server) handleAdminListDokument(w http.ResponseWriter, r *http.Request) {
	arbeiterFilter, ok := parseOptionalUUIDQuery(w, r, "arbeiter")
	if !ok {
		return
	}
	rows, err := s.queries.AdminListDokument(r.Context(), arbeiterFilter)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	out := make([]dokumentResponse, 0, len(rows))
	for _, d := range rows {
		out = append(out, toDokumentResponse(d))
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

// handleDownloadDokument streams a document. Worker may fetch only their own;
// admin may fetch any. A foreign worker access returns 404 (no existence leak).
func (s *Server) handleDownloadDokument(w http.ResponseWriter, r *http.Request) {
	ident, _ := identityFrom(r.Context())
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	dok, err := s.queries.GetDokumentByID(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Dokument nicht gefunden.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if dok.ArbeiterID != ident.ArbeiterID && ident.Rolle != "admin" {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Dokument nicht gefunden.")
		return
	}
	rc, err := s.store.Open(r.Context(), dok.StorageKey)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	defer func() { _ = rc.Close() }()
	w.Header().Set("Content-Type", dok.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", dok.Dateiname))
	if dok.Groesse > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(dok.Groesse, 10))
	}
	_, _ = io.Copy(w, rc)
}
