package server

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/db"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

type ueberstundenResponse struct {
	ArbeiterID      uuid.UUID `json:"arbeiter_id"`
	Name            string    `json:"name,omitempty"`
	Jahr            int       `json:"jahr"`
	Monat           int       `json:"monat"`
	Werktage        int       `json:"werktage"`
	UrlaubKrankTage int       `json:"urlaub_krank_tage"`
	SollMinuten     int64     `json:"soll_minuten"`
	IstMinuten      int64     `json:"ist_minuten"`
	SaldoMinuten    int64     `json:"saldo_minuten"`
}

// monthBounds returns [start, end) of the given month in UTC.
func monthBounds(jahr, monat int) (start, end time.Time) {
	start = time.Date(jahr, time.Month(monat), 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 1, 0)
}

// countWeekdays counts Mon–Fri days in [start, end). Public holidays are NOT
// excluded (§13 "Feiertage später").
func countWeekdays(start, end time.Time) int {
	n := 0
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			n++
		}
	}
	return n
}

// creditedWorkdays counts distinct Mon–Fri days within [start, end) covered by the
// approved-leave rows (one arbeiter). bis_datum is inclusive. Overlapping requests
// do not double-count.
func creditedWorkdays(rows []db.ListGenehmigteAbwesenheitRow, start, end time.Time) int {
	covered := make(map[string]struct{})
	for _, row := range rows {
		from := row.VonDatum.UTC()
		if from.Before(start) {
			from = start
		}
		to := row.BisDatum.UTC()
		for d := from; !d.After(to) && d.Before(end); d = d.AddDate(0, 0, 1) {
			if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
				covered[d.Format(dateFormat)] = struct{}{}
			}
		}
	}
	return len(covered)
}

// sollMinuten = (wochenstunden/5)*60 * effektiveWerktage, rounded to whole minutes.
func sollMinuten(wochenstunden string, effektiveWerktage int) int64 {
	wh, err := strconv.ParseFloat(wochenstunden, 64)
	if err != nil {
		wh = 0
	}
	return int64(math.Round(wh / 5.0 * 60.0 * float64(effektiveWerktage)))
}

// parseMonat reads ?jahr=&monat=; defaults to the current month if both absent.
func parseMonat(w http.ResponseWriter, r *http.Request) (jahr, monat int, ok bool) {
	js, ms := r.URL.Query().Get("jahr"), r.URL.Query().Get("monat")
	if js == "" && ms == "" {
		now := time.Now().UTC()
		return now.Year(), int(now.Month()), true
	}
	if js == "" || ms == "" {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "jahr und monat zusammen angeben.")
		return 0, 0, false
	}
	jahr, err := strconv.Atoi(js)
	if err != nil || jahr < 2000 || jahr > 2100 {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültiges jahr.")
		return 0, 0, false
	}
	monat, err = strconv.Atoi(ms)
	if err != nil || monat < 1 || monat > 12 {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültiger monat (1-12).")
		return 0, 0, false
	}
	return jahr, monat, true
}

// computeUeberstunden assembles the balance for one arbeiter/month.
func computeUeberstunden(id uuid.UUID, name string, jahr, monat int, start, end time.Time,
	wochenstunden string, ist int64, abw []db.ListGenehmigteAbwesenheitRow) ueberstundenResponse {
	werktage := countWeekdays(start, end)
	credited := creditedWorkdays(abw, start, end)
	effektiv := werktage - credited
	if effektiv < 0 {
		effektiv = 0
	}
	soll := sollMinuten(wochenstunden, effektiv)
	return ueberstundenResponse{
		ArbeiterID:      id,
		Name:            name,
		Jahr:            jahr,
		Monat:           monat,
		Werktage:        werktage,
		UrlaubKrankTage: credited,
		SollMinuten:     soll,
		IstMinuten:      ist,
		SaldoMinuten:    ist - soll,
	}
}

// handleOwnUeberstunden returns the authenticated worker's balance for a month.
func (s *Server) handleOwnUeberstunden(w http.ResponseWriter, r *http.Request) {
	ident, _ := identityFrom(r.Context())
	jahr, monat, ok := parseMonat(w, r)
	if !ok {
		return
	}
	start, end := monthBounds(jahr, monat)

	a, err := s.queries.GetArbeiterByID(r.Context(), ident.ArbeiterID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	ist, err := s.queries.AdminSumZeitbuchung(r.Context(), db.AdminSumZeitbuchungParams{
		ArbeiterID: &ident.ArbeiterID, Von: &start, Bis: &end,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	lastDay := end.AddDate(0, 0, -1)
	abw, err := s.queries.ListGenehmigteAbwesenheit(r.Context(), db.ListGenehmigteAbwesenheitParams{
		Von: start, Bis: lastDay, ArbeiterID: &ident.ArbeiterID,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK,
		computeUeberstunden(a.ID, "", jahr, monat, start, end, a.Wochenstunden, ist.SummeMinuten, abw))
}

// handleAdminUeberstunden returns balances for all active workers (or one via ?arbeiter=).
func (s *Server) handleAdminUeberstunden(w http.ResponseWriter, r *http.Request) {
	jahr, monat, ok := parseMonat(w, r)
	if !ok {
		return
	}
	arbeiterFilter, ok := parseOptionalUUIDQuery(w, r, "arbeiter")
	if !ok {
		return
	}
	start, end := monthBounds(jahr, monat)
	lastDay := end.AddDate(0, 0, -1)

	var arbeiterListe []db.Arbeiter
	if arbeiterFilter != nil {
		a, err := s.queries.GetArbeiterByID(r.Context(), *arbeiterFilter)
		if errors.Is(err, pgx.ErrNoRows) {
			platform.WriteError(w, http.StatusNotFound, "not_found", "Arbeiter nicht gefunden.")
			return
		}
		if err != nil {
			s.serverError(w, r, err)
			return
		}
		arbeiterListe = []db.Arbeiter{a}
	} else {
		aktiv := true
		rows, err := s.queries.ListArbeiter(r.Context(), &aktiv)
		if err != nil {
			s.serverError(w, r, err)
			return
		}
		arbeiterListe = rows
	}

	perZeit, err := s.queries.AdminSumZeitbuchungPerArbeiter(r.Context(), db.AdminSumZeitbuchungPerArbeiterParams{
		ArbeiterID: arbeiterFilter, Von: &start, Bis: &end,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	istMap := make(map[uuid.UUID]int64, len(perZeit))
	for _, p := range perZeit {
		istMap[p.ArbeiterID] = p.SummeMinuten
	}

	abw, err := s.queries.ListGenehmigteAbwesenheit(r.Context(), db.ListGenehmigteAbwesenheitParams{
		Von: start, Bis: lastDay, ArbeiterID: arbeiterFilter,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	abwMap := make(map[uuid.UUID][]db.ListGenehmigteAbwesenheitRow)
	for _, row := range abw {
		abwMap[row.ArbeiterID] = append(abwMap[row.ArbeiterID], row)
	}

	out := make([]ueberstundenResponse, 0, len(arbeiterListe))
	for _, a := range arbeiterListe {
		out = append(out, computeUeberstunden(a.ID, a.Name, jahr, monat, start, end,
			a.Wochenstunden, istMap[a.ID], abwMap[a.ID]))
	}
	platform.WriteJSON(w, http.StatusOK, out)
}
