# Testplanung — Codequality & Unittests

**Projekt:** Prekaj-Zeiterfassung · **Datum:** 2026-06-13 · **Gruppengröße:** 1 (solo)

## 1. Aufgabe

1. Gängiges Codequality-Werkzeug einsetzen und eine **Komplexitätsanalyse** des
   bestehenden Codes durchführen.
2. Für **eine** als komplex ermittelte Komponente (1 pro Gruppenmitglied → solo = 1)
   ein Set **Testfälle** erstellen.
3. **Testziel: Entscheidungsüberdeckung (= Zweigüberdeckung / decision coverage).**
4. Testfälle durchführen.
5. Export: Unittests als Text inkl. Durchführungsbericht + Code-Quality-Bericht.

## 2. Werkzeugauswahl

| Zweck | Werkzeug | Begründung |
|---|---|---|
| Code-Quality + Komplexität (zyklomatisch & kognitiv) + Coverage-Visualisierung | **SonarQube Community** (lokal in Docker) | gängiges Standardwerkzeug (siehe Aufgabenlink); eingebaute Go-Unterstützung; misst Cyclomatic & Cognitive Complexity je Funktion/Datei; importiert Go-Coverage |
| Unittests | **Go-Standard `testing`** (table-driven) | nativer Teststandard des Backends, keine Zusatzabhängigkeit, deckt unexportierte Funktionen ab (White-Box) |
| Coverage-Messung | **`go test -cover` / `go tool cover`** | erzeugt Coverage-Profil (`coverage.out`) + HTML-Nachweis; Profil wird von SonarQube eingelesen |

> SonarQube läuft lokal: `docker run -d --name sonarqube -p 9000:9000 sonarqube:community`,
> Weboberfläche unter http://localhost:9000.

## 3. Komplexitätsanalyse → Komponentenauswahl

Das Backend (Go) enthält die gesamte Geschäftslogik. SonarQube bewertet die zyklomatische
und kognitive Komplexität je Funktion. Die fachlich komplexeste, **ohne Datenbank**
testbare Komponente ist die **Überstunden-Berechnung** in
`backend/internal/server/ueberstunden.go`. Sie kombiniert Schleifen, Datumsarithmetik,
Überlappungs-Deduplizierung und umfangreiche Eingabevalidierung.

Gewählte Komponente (solo): **Überstunden-Berechnung — `ueberstunden.go`**.

Reine Funktionen (kein DB-Zugriff → vollständig unit-testbar):

| Funktion | Zyklomat. Komplexität (ca.) | Rolle |
|---|---|---|
| `parseMonat` | hoch (~8–10) | Query-Parameter `jahr`/`monat` prüfen |
| `creditedWorkdays` | ~5 | gutgeschriebene Werktage aus genehmigten Abwesenheiten |
| `countWeekdays` | ~3 | Werktage (Mo–Fr) im Monat zählen |
| `sollMinuten` | ~2 | Soll-Minuten aus Wochenstunden |
| `computeUeberstunden` | ~2 | Saldo = Ist − Soll zusammensetzen |
| `monthBounds` | 1 | Monatsgrenzen [start, end) |

> Die HTTP-Handler `handleAdminUeberstunden` / `handlePatchZeit` haben laut SonarQube die
> höchste Komplexität, benötigen für Tests aber DB-Mocking (Interface-Refactoring).
> Für eine saubere Zweigüberdeckung sind die reinen Funktionen besser geeignet; die
> Handler bleiben hier außen vor (out of scope der Übung).

## 4. Vorgehen Entscheidungsüberdeckung

Die zyklomatische Komplexität entspricht der Zahl unabhängiger Pfade → pro Funktion
werden mindestens so viele Testfälle entworfen, dass **jede Entscheidung beide Ausgänge
(true/false)** durchläuft. Die vollständige Zuordnung Testfall → Zweig steht in
[`testfaelle.md`](./testfaelle.md). Nachweis über `go tool cover` (HTML, jeder Block grün)
und SonarQube-Coverage auf Dateiebene.

## 5. Artefakte / Exporte

| Artefakt | Pfad |
|---|---|
| Unittests (Quellcode) | `backend/internal/server/ueberstunden_test.go` |
| Testfälle (Tabelle) | `docs/testing/testfaelle.md` |
| Durchführungsbericht (Text) | `docs/testing/durchfuehrungsbericht.txt` |
| Coverage-Profil / HTML | `backend/coverage.out` / `backend/coverage.html` |
| Test-Report (für SonarQube) | `backend/test-report.json` |
| SonarQube-Metriken (Export) | `docs/testing/sonarqube-metriken.json` |
| SonarQube-Quality-Bericht | Screenshots (Overview, Measures→Complexity, Coverage) |

## 6. Befehle

```bash
# Tests + Coverage (aus backend/)
make test         # go test -v der Komponente
make coverage     # erzeugt coverage.out, coverage.html, test-report.json, Bericht
make sonar TOKEN=<sonar-token>   # Scanner gegen http://localhost:9000
```
