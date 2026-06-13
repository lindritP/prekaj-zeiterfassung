# Abgabe — Codequality & Unittests

Komponente: **Überstunden-Berechnung** (`backend/internal/server/ueberstunden.go`)
Testziel: **Entscheidungsüberdeckung** · Werkzeuge: **SonarQube** + **Go test/cover**

## Inhalt / Abgabe-Artefakte

| Datei | Inhalt |
|---|---|
| `testplanung.md` | Werkzeugauswahl, Vorgehen, Komponentenauswahl |
| `testfaelle.md` | 24 Testfälle (ID, Eingabe, Erwartung, abgedeckter Zweig) |
| `durchfuehrungsbericht.txt` | `go test -v`-Lauf + Coverage je Funktion (alle PASS) |
| `codequality-bericht.md` | SonarQube-Ergebnisse (Komplexität, Findings, Coverage) |
| `sonarqube-projekt-metriken.json` | Werkzeug-Export: Projekt-Kennzahlen |
| `sonarqube-ueberstunden-metriken.json` | Werkzeug-Export: Datei-Kennzahlen |
| `sonarqube-issues.json` | Werkzeug-Export: alle Findings |
| `../../backend/coverage.html` | visueller Coverage-Nachweis (jeder Block grün) |
| `../../backend/internal/server/ueberstunden_test.go` | Unittest-Quellcode |

## Ergebnis in einem Satz

24 Testfälle, **alle PASS**; die 6 reinen Funktionen der Komponente erreichen 100 %
Coverage (1 nachweislich unerreichbarer Defensiv-Zweig dokumentiert) → Entscheidungs-
überdeckung erfüllt. SonarQube: 0 Bugs, 0 Vulnerabilities, 7 Maintainability-Smells.

## Noch zu erledigen: Screenshots (für den Quality-Bericht)

SonarQube läuft unter http://localhost:9000 (Projekt `prekaj-zeiterfassung-backend`).
Bitte 3 Screenshots ablegen (z. B. hier als `screenshot-*.png`):

1. **Overview** — Dashboard mit Quality Gate, Bugs/Smells/Coverage/Complexity.
2. **Measures → Complexity** — Cyclomatic/Cognitive Complexity je Datei (Ranking).
3. **Code → `ueberstunden.go` → Coverage** — Coverage-Ansicht der Datei.

## Reproduktion

```bash
docker start sonarqube
make coverage                          # Tests + Coverage + Berichte
make sonar TOKEN=<analysis-token>      # SonarQube-Scan
```
