package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/db"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/pdf"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

// viennaLoc is the display timezone for reports (storage stays UTC). Embedded
// tzdata (see cmd/api) guarantees this loads even in a distroless container.
var viennaLoc = mustLoadVienna()

func mustLoadVienna() *time.Location {
	loc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		return time.UTC
	}
	return loc
}

// handleMonatsbericht (admin) generates the monthly PDF report for one arbeiter.
func (s *Server) handleMonatsbericht(w http.ResponseWriter, r *http.Request) {
	arbeiterID, ok := parseOptionalUUIDQuery(w, r, "arbeiter")
	if !ok {
		return
	}
	if arbeiterID == nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Parameter arbeiter erforderlich.")
		return
	}
	jahr, monat, ok := parseMonat(w, r)
	if !ok {
		return
	}
	start, end := monthBounds(jahr, monat)
	lastDay := end.AddDate(0, 0, -1)

	a, err := s.queries.GetArbeiterByID(r.Context(), *arbeiterID)
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusNotFound, "not_found", "Arbeiter nicht gefunden.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	zb, err := s.queries.AdminListZeitbuchung(r.Context(), db.AdminListZeitbuchungParams{
		ArbeiterID: arbeiterID, Von: &start, Bis: &end,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	bsRows, err := s.queries.ListBaustellen(r.Context(), nil)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	bsName := make(map[uuid.UUID]string, len(bsRows))
	for _, b := range bsRows {
		bsName[b.ID] = b.Name
	}
	ist, err := s.queries.AdminSumZeitbuchung(r.Context(), db.AdminSumZeitbuchungParams{
		ArbeiterID: arbeiterID, Von: &start, Bis: &end,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	abw, err := s.queries.ListGenehmigteAbwesenheit(r.Context(), db.ListGenehmigteAbwesenheitParams{
		Von: start, Bis: lastDay, ArbeiterID: arbeiterID,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	u := computeUeberstunden(a.ID, a.Name, jahr, monat, start, end, a.Wochenstunden, ist.SummeMinuten, abw)

	zeilen := make([]pdf.BerichtZeile, 0, len(zb))
	for _, z := range zb {
		baustelle := "—"
		if z.BaustelleID != nil {
			if n, found := bsName[*z.BaustelleID]; found {
				baustelle = n
			}
		}
		ende, dauer := "läuft", "—"
		if z.EndZeit != nil {
			ende = z.EndZeit.In(viennaLoc).Format("15:04")
			dm := dauerMinuten(z.StartZeit, *z.EndZeit, z.PauseMinuten)
			dauer = fmt.Sprintf("%d:%02d", dm/60, dm%60)
		}
		zeilen = append(zeilen, pdf.BerichtZeile{
			Datum:     z.StartZeit.In(viennaLoc).Format("02.01.2006"),
			Baustelle: baustelle,
			Start:     z.StartZeit.In(viennaLoc).Format("15:04"),
			Ende:      ende,
			PauseMin:  z.PauseMinuten,
			Dauer:     dauer,
		})
	}

	pdfBytes, err := pdf.Monatsbericht(pdf.MonatsberichtData{
		ArbeiterName:    a.Name,
		Jahr:            jahr,
		Monat:           monat,
		Zeilen:          zeilen,
		IstMinuten:      u.IstMinuten,
		SollMinuten:     u.SollMinuten,
		SaldoMinuten:    u.SaldoMinuten,
		UrlaubKrankTage: u.UrlaubKrankTage,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("inline; filename=%q", fmt.Sprintf("Monatsbericht_%d-%02d.pdf", jahr, monat)))
	_, _ = w.Write(pdfBytes)
}
