# Code-Quality-Bericht — SonarQube

**Werkzeug:** SonarQube Community Edition 26.6 (lokal in Docker, http://localhost:9000)
**Projekt:** `prekaj-zeiterfassung-backend` · **Sprache:** Go · **Datum:** 2026-06-13
**Scanner:** sonar-scanner (aus `backend/`, Konfiguration `backend/sonar-project.properties`)

Rohdaten-Export (Werkzeug-Export): `sonarqube-projekt-metriken.json`,
`sonarqube-ueberstunden-metriken.json`, `sonarqube-issues.json` (in diesem Ordner).
Zusätzlich Screenshots der Weboberfläche (Overview / Measures→Complexity / Code→Coverage).

## 1. Projekt-Kennzahlen

| Metrik | Wert |
|---|---|
| Lines of Code (Go) | 2.339 |
| Funktionen | 101 |
| **Cyclomatic Complexity (gesamt)** | **397** |
| **Cognitive Complexity (gesamt)** | **312** |
| Bugs | 0 |
| Vulnerabilities | 0 |
| Code Smells | 7 |
| Duplications | 0,0 % |
| Coverage (gesamt) | 4,2 % |

> Hinweis: Generierter Code (`*.sql.go`, `internal/db/**`), Tests und HTML-Reports sind
> von der Analyse ausgeschlossen, damit die Metriken nur den handgeschriebenen Code abbilden.
> Die niedrige Gesamt-Coverage ist erwartet: getestet wird gezielt **eine** Komponente
> (Übungsumfang), nicht das ganze Backend.

## 2. Komplexitätsanalyse — Ranking der Dateien

Ermittelt über die SonarQube-Metriken `complexity` (zyklomatisch) und
`cognitive_complexity` je Datei:

| Rang | Datei | Cyclomatic | Cognitive | Bemerkung |
|---|---|---|---|---|
| 1 | `internal/server/zeitbuchung.go` | **61** | 52 | komplexeste Datei; Komplexität v. a. in DB-Handlern |
| 2 | `internal/server/ueberstunden.go` | **44** | 43 | **gewählte Komponente** (komplexeste *reine* Logik) |
| 3 | `internal/server/urlaubsantrag.go` | 40 | 35 | |
| 4 | `internal/server/dokument.go` | 38 | 31 | |
| 5 | `internal/server/arbeiter.go` | 29 | 24 | |
| 5 | `internal/server/auth.go` | 29 | 24 | |
| 7 | `internal/server/bericht.go` | 18 | 20 | |
| 8 | `internal/pdf/monatsbericht.go` | 8 | 5 | |
| 9 | `internal/auth/jwt.go` | 7 | 5 | |

### Komponentenauswahl (1 Komponente, solo)

Gewählt: **`ueberstunden.go` (Überstunden-Berechnung)** — Rang 2 der Komplexität.
`zeitbuchung.go` (Rang 1) konzentriert seine Komplexität in **datenbankgebundenen
Handlern** (`handleStartZeit`/`handleStopZeit`/`handlePatchZeit`), deren Test ein
DB-Mocking/Interface-Refactoring erfordern würde. `ueberstunden.go` enthält dagegen die
**komplexeste rein funktionale Logik** (Datumsarithmetik, Überlappungs-Dedup, umfangreiche
Eingabevalidierung in `parseMonat`) — ideal, um **Entscheidungsüberdeckung** sauber und
ohne DB nachzuweisen.

## 3. Qualitäts-Findings (7 Code Smells, 0 Bugs)

| Datei:Zeile | Regel | Befund |
|---|---|---|
| `zeitbuchung.go:248` | go:S3776 | `handlePatchZeit` Cognitive Complexity 18 > 15 |
| `bericht.go:30` | go:S3776 | Methode Cognitive Complexity 19 > 15 |
| `dokument.go:46` | go:S3776 | Methode Cognitive Complexity 19 > 15 |
| `ueberstunden.go:101` | go:S107 | `computeUeberstunden` hat 9 Parameter > 7 |
| `zeitbuchung.go:40` | go:S1192 | dupliziertes Literal „end_zeit muss nach start_zeit liegen." (3×) |
| `arbeiter.go:125` | go:S1192 | dupliziertes Literal „Arbeiter nicht gefunden." (3×) |
| `baustelle.go:92` | go:S1192 | dupliziertes Literal „Baustelle nicht gefunden." (3×) |

Alle Findings sind „Maintainability"-Hinweise (keine Bugs/Sicherheitslücken). Empfehlung:
Fehlermeldungs-Literale als Konstanten zentralisieren und die drei komplexesten Handler
in Hilfsfunktionen aufteilen.

## 4. Coverage der gewählten Komponente

| Metrik (`ueberstunden.go`) | Wert |
|---|---|
| Cyclomatic Complexity | 44 |
| Cognitive Complexity | 43 |
| Coverage (Datei) | 41,6 % |
| Funktionen | 8 |

Die 41,6 % ergeben sich, weil `ueberstunden.go` neben den 6 getesteten reinen Funktionen
auch zwei **DB-Handler** enthält (`handleOwnUeberstunden`, `handleAdminUeberstunden`),
die bewusst nicht Teil des Tests sind. Die getesteten reinen Funktionen erreichen laut
`go tool cover -func` **100 %** (Ausnahme: ein nachweislich unerreichbarer Defensiv-Zweig,
siehe `testfaelle.md`).

### Hinweis Entscheidungsüberdeckung vs. SonarQube

Go erzeugt ein **statement-/zeilenbasiertes** Coverage-Profil (`coverage.out`); SonarQube
zeigt daraus **Line Coverage**. Die geforderte **Entscheidungsüberdeckung** wird daher
zusätzlich belegt durch (a) die Testfall→Zweig-Matrix in `testfaelle.md` und (b) den
HTML-Coverage-Nachweis `backend/coverage.html`, in dem jeder Block der Zielfunktionen als
abgedeckt (grün) markiert ist. Die zyklomatische Komplexität aus SonarQube diente dabei
als Untergrenze für die Anzahl nötiger Testfälle pro Funktion.

## 5. Reproduktion

```bash
docker start sonarqube                     # http://localhost:9000
cd backend
make -C .. coverage                        # erzeugt coverage.out + test-report.json
make -C .. sonar TOKEN=<analysis-token>    # bzw. sonar-scanner direkt
```
