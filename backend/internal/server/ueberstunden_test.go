package server

// Testfälle für die komplexe Komponente "Überstunden-Berechnung"
// (siehe SonarQube-Komplexitätsanalyse, docs/testing/).
//
// Testziel: ENTSCHEIDUNGSÜBERDECKUNG (= Zweigüberdeckung / decision coverage) —
// jede if-/Schleifen-Bedingung wird mit beiden Ausgängen (true/false) durchlaufen.
// Werkzeug: Go-Standard-Testing + `go test -cover` (Profil für SonarQube).
//
// Wochentags-Anker für die Datumsfälle: 2023-01-01 = Sonntag, 2023-01-02 = Montag.
// Januar 2023 hat 22 Werktage (Mo–Fr), monthBounds(2023,1) = [2023-01-01, 2023-02-01).
//
// Abdeckungs-Matrix (Funktion → Zweig → Testfall-ID):
//   monthBounds        : normaler Monat (MB1) / Jahreswechsel Dez→Jan (MB2)
//   countWeekdays      : Werktag gezählt (CW1/CW2/CW5) / Wochenende übersprungen (CW3)
//                        / leerer Bereich, Schleife 0× (CW4)
//   creditedWorkdays   : keine Zeilen, Schleife 0× (CR6) / from<start clamp true (CR2)
//                        / clamp false (CR1) / bis inklusiv & end exklusiv (CR3)
//                        / Wochenende übersprungen (CR5) / Überlappung dedupliziert (CR4)
//   sollMinuten        : ParseFloat ok (SM1/SM2/SM5) / Rundung (SM2)
//                        / ParseFloat-Fehler→0 (SM3/SM4)
//   computeUeberstunden: effektiv>=0 Normalpfad (CU1/CU2); der Zweig effektiv<0 ist
//                        unter der Invariante credited<=werktage nachweislich
//                        unerreichbar (dokumentiert in docs/testing/testfaelle.md).
//   parseMonat         : beide Parameter fehlen→now (PM1) / genau einer fehlt→400
//                        (PM2/PM3) / jahr nicht numerisch→400 (PM4) / jahr<2000 (PM5)
//                        / jahr>2100 (PM6) / monat nicht numerisch→400 (PM7)
//                        / monat<1 (PM8) / monat>12 (PM9) / gültig (PM10)
//                        / Grenzwerte gültig (PM11/PM12)

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/db"
)

// d ist ein Kurzhelfer für ein UTC-Datum (Mitternacht).
func d(jahr, monat, tag int) time.Time {
	return time.Date(jahr, time.Month(monat), tag, 0, 0, 0, 0, time.UTC)
}

func TestMonthBounds(t *testing.T) {
	tests := []struct {
		name               string
		jahr, monat        int
		wantStart, wantEnd time.Time
	}{
		{"MB1 normaler Monat", 2023, 1, d(2023, 1, 1), d(2023, 2, 1)},
		{"MB2 Jahreswechsel Dez", 2026, 12, d(2026, 12, 1), d(2027, 1, 1)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end := monthBounds(tc.jahr, tc.monat)
			if !start.Equal(tc.wantStart) || !end.Equal(tc.wantEnd) {
				t.Errorf("monthBounds(%d,%d) = [%s, %s); want [%s, %s)",
					tc.jahr, tc.monat, start, end, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

func TestCountWeekdays(t *testing.T) {
	tests := []struct {
		name       string
		start, end time.Time
		want       int
	}{
		{"CW1 ganzer Januar 2023", d(2023, 1, 1), d(2023, 2, 1), 22},
		{"CW2 eine Woche Mo-Mo", d(2023, 1, 2), d(2023, 1, 9), 5},
		{"CW3 nur Wochenende Sa-Mo", d(2023, 1, 7), d(2023, 1, 9), 0},
		{"CW4 leerer Bereich", d(2023, 1, 2), d(2023, 1, 2), 0},
		{"CW5 ein Werktag Mo-Di", d(2023, 1, 2), d(2023, 1, 3), 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := countWeekdays(tc.start, tc.end); got != tc.want {
				t.Errorf("countWeekdays(%s, %s) = %d; want %d", tc.start, tc.end, got, tc.want)
			}
		})
	}
}

// abw baut eine Abwesenheits-Zeile (Datums-only, UTC).
func abw(von, bis time.Time) db.ListGenehmigteAbwesenheitRow {
	return db.ListGenehmigteAbwesenheitRow{VonDatum: von, BisDatum: bis}
}

func TestCreditedWorkdays(t *testing.T) {
	start, end := monthBounds(2023, 1) // [2023-01-01, 2023-02-01)
	tests := []struct {
		name string
		rows []db.ListGenehmigteAbwesenheitRow
		want int
	}{
		{"CR1 Zeile komplett im Monat Mo-Fr", []db.ListGenehmigteAbwesenheitRow{abw(d(2023, 1, 2), d(2023, 1, 6))}, 5},
		{"CR2 Beginn vor Monat -> clamp", []db.ListGenehmigteAbwesenheitRow{abw(d(2022, 12, 28), d(2023, 1, 3))}, 2},
		{"CR3 Ende nach Monat -> end-exklusiv", []db.ListGenehmigteAbwesenheitRow{abw(d(2023, 1, 30), d(2023, 2, 10))}, 2},
		{"CR4 Überlappung dedupliziert", []db.ListGenehmigteAbwesenheitRow{
			abw(d(2023, 1, 2), d(2023, 1, 4)),
			abw(d(2023, 1, 3), d(2023, 1, 5)),
		}, 4},
		{"CR5 nur Wochenende", []db.ListGenehmigteAbwesenheitRow{abw(d(2023, 1, 7), d(2023, 1, 8))}, 0},
		{"CR6 keine Zeilen", nil, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := creditedWorkdays(tc.rows, start, end); got != tc.want {
				t.Errorf("creditedWorkdays(%s) = %d; want %d", tc.name, got, tc.want)
			}
		})
	}
}

func TestSollMinuten(t *testing.T) {
	tests := []struct {
		name          string
		wochenstunden string
		tage          int
		want          int64
	}{
		{"SM1 40h, 22 Tage", "40", 22, 10560},
		{"SM2 Rundung 40.3h, 1 Tag", "40.3", 1, 484}, // 40.3/5*60 = 483.6 -> 484
		{"SM3 unparsbar -> 0", "abc", 22, 0},
		{"SM4 leer -> 0", "", 5, 0},
		{"SM5 0 Tage -> 0", "40", 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sollMinuten(tc.wochenstunden, tc.tage); got != tc.want {
				t.Errorf("sollMinuten(%q, %d) = %d; want %d", tc.wochenstunden, tc.tage, got, tc.want)
			}
		})
	}
}

func TestComputeUeberstunden(t *testing.T) {
	start, end := monthBounds(2023, 1) // 22 Werktage

	t.Run("CU1 mit Abwesenheit, positives Saldo", func(t *testing.T) {
		rows := []db.ListGenehmigteAbwesenheitRow{abw(d(2023, 1, 2), d(2023, 1, 6))} // 5 Werktage gutgeschrieben
		got := computeUeberstunden(uuid.Nil, "Test", 2023, 1, start, end, "40", 9000, rows)
		// effektiv = 22 - 5 = 17; soll = 8*60*17 = 8160; saldo = 9000 - 8160 = 840
		if got.Werktage != 22 || got.UrlaubKrankTage != 5 || got.SollMinuten != 8160 ||
			got.IstMinuten != 9000 || got.SaldoMinuten != 840 || got.Jahr != 2023 || got.Monat != 1 {
			t.Errorf("CU1 unerwartet: %+v", got)
		}
	})

	t.Run("CU2 ohne Abwesenheit, negatives Saldo (Minusstunden)", func(t *testing.T) {
		got := computeUeberstunden(uuid.Nil, "", 2023, 1, start, end, "40", 0, nil)
		// soll = 8*60*22 = 10560; saldo = 0 - 10560 = -10560
		if got.SollMinuten != 10560 || got.SaldoMinuten != -10560 || got.UrlaubKrankTage != 0 {
			t.Errorf("CU2 unerwartet: %+v", got)
		}
	})
}

func TestParseMonat(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name       string
		query      string
		wantOK     bool
		wantStatus int // nur relevant wenn wantOK == false
		wantJahr   int // nur relevant wenn wantOK == true (0 = nicht prüfen)
		wantMonat  int
	}{
		{"PM1 beide fehlen -> jetzt", "", true, 0, now.Year(), int(now.Month())},
		{"PM2 monat fehlt -> 400", "jahr=2026", false, http.StatusBadRequest, 0, 0},
		{"PM3 jahr fehlt -> 400", "monat=6", false, http.StatusBadRequest, 0, 0},
		{"PM4 jahr nicht numerisch -> 400", "jahr=abc&monat=6", false, http.StatusBadRequest, 0, 0},
		{"PM5 jahr < 2000 -> 400", "jahr=1999&monat=6", false, http.StatusBadRequest, 0, 0},
		{"PM6 jahr > 2100 -> 400", "jahr=2101&monat=6", false, http.StatusBadRequest, 0, 0},
		{"PM7 monat nicht numerisch -> 400", "jahr=2026&monat=xx", false, http.StatusBadRequest, 0, 0},
		{"PM8 monat < 1 -> 400", "jahr=2026&monat=0", false, http.StatusBadRequest, 0, 0},
		{"PM9 monat > 12 -> 400", "jahr=2026&monat=13", false, http.StatusBadRequest, 0, 0},
		{"PM10 gültig", "jahr=2026&monat=6", true, 0, 2026, 6},
		{"PM11 Grenzwert min gültig", "jahr=2000&monat=1", true, 0, 2000, 1},
		{"PM12 Grenzwert max gültig", "jahr=2100&monat=12", true, 0, 2100, 12},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?"+tc.query, nil)
			rec := httptest.NewRecorder()
			jahr, monat, ok := parseMonat(rec, req)
			if ok != tc.wantOK {
				t.Fatalf("parseMonat(%q) ok = %v; want %v", tc.query, ok, tc.wantOK)
			}
			if !tc.wantOK {
				if rec.Code != tc.wantStatus {
					t.Errorf("parseMonat(%q) status = %d; want %d", tc.query, rec.Code, tc.wantStatus)
				}
				return
			}
			if jahr != tc.wantJahr || monat != tc.wantMonat {
				t.Errorf("parseMonat(%q) = (%d,%d); want (%d,%d)", tc.query, jahr, monat, tc.wantJahr, tc.wantMonat)
			}
		})
	}
}
