# Abgabe — Aufgabe 4: Spezifikationsbasierter Testfallentwurf & Durchführung

**Anwendungsfall:** *Zeiterfassung starten / beenden* · **Methodik:** Black-Box / spezifikationsbasiert
(Äquivalenzklassen, Grenzwertanalyse, Entscheidungstabelle, Zustandsübergang, Anwendungsfalltest) ·
**Werkzeuge:** Postman-Collection + Newman gegen die laufende REST-API.

## Ergebnis in einem Satz

31 spezifikationsbasiert entworfene Testfälle wurden gegen die laufende API ausgeführt — **29
erfolgreich, 2 fehlerhaft** (BUG-01: `end_zeit` in der Zukunft wird akzeptiert; BUG-02: Pause auf
laufender Buchung wird akzeptiert) — alle Ergebnisse stammen aus einem realen Newman-Lauf.

## Inhalt / Abgabe-Artefakte

| Datei / Ordner | Inhalt |
|---|---|
| `testkonzept-aufgabe4.md` | Anwendungsfall-Auswahl, Testbasis, Herleitung der Methoden (ÄK/GW/ET/ZÜ/AF), Rückverfolgbarkeit zu Aufgabe 2 |
| `testfaelle-aufgabe4.md` | **Haupt-Deliverable:** Tabelle 1 (Entwurf, 8+ Attribute) + Tabelle 2 (Ausführung, 7 Attribute), verknüpft über die Testfall-ID |
| `postman/zeiterfassung.postman_collection.json` | ausführbare Collection (Setup + 31 Testfälle mit Assertions) |
| `postman/local.postman_environment.json` | Environment (baseUrl, Seed-Admin) |
| `postman/build-collection.py` | reproduzierbarer Generator der Collection (dokumentiert die Testlogik kompakt) |
| `durchfuehrungsbericht/newman-summary.txt` | Konsolen-Zusammenfassung des realen Laufs |
| `durchfuehrungsbericht/newman-report.json` | Maschinen-Report (Newman) |
| `durchfuehrungsbericht/newman-report.html` | visueller Report (htmlextra) |
| `bugs/BUG-01-future-end-zeit.sh` + `BUG-01.txt` + `BUG-01.png` | Ein-Befehl-Repro + Transcript + **Screenshot** zu BUG-01 |
| `bugs/BUG-02-pause-auf-laufender-buchung.sh` + `BUG-02.txt` + `BUG-02.png` | Ein-Befehl-Repro + Transcript + **Screenshot** zu BUG-02 |
| `Aufgabe4_Pruefbericht.docx` | **Prüfbericht (Word)** — Deckblatt, Inhaltsverzeichnis, 11 nummerierte Kapitel, formatierte/farbcodierte Testfall-Tabellen mit allen Pflichtspalten, eingebettete Screenshots — die Abgabe-Datei |

## Gefundene Fehler

| Bug | Kurzbeschreibung | Testfall | Schwere |
|---|---|---|---|
| **BUG-01** | `end_zeit` in der Zukunft (z. B. 2030) wird bei `stop` und `PATCH` akzeptiert → Dauer ≈ 1.290 Tage. `start_zeit` wird gegen Zukunft geprüft, `end_zeit` nicht. | TC-STOP-07 | mittel–hoch |
| **BUG-02** | `pause_minuten` auf einer **laufenden** Buchung wird akzeptiert (200) und beim Stop still verworfen. | TC-PATCH-09 | niedrig |

## Reproduktion

```bash
# 1) Stack starten (Docker Desktop muss laufen)
make db-up && make migrate-up && make seed && make run-api    # API auf :8080

# 2) Testfälle ausführen (aus docs/testing/aufgabe4/)
newman run postman/zeiterfassung.postman_collection.json \
  -e postman/local.postman_environment.json \
  -r cli,json,htmlextra \
  --reporter-json-export durchfuehrungsbericht/newman-report.json \
  --reporter-htmlextra-export durchfuehrungsbericht/newman-report.html
# erwartet: 88 Assertions, 2 Fehlschläge (TC-STOP-07 → BUG-01, TC-PATCH-09 → BUG-02)

# 3) Collection bei Bedarf neu generieren
python3 postman/build-collection.py
```

## Abgabe-Dokument (Word, mit eingebetteten Screenshots)

Die Abgabe ist **`Aufgabe4_Pruefbericht.docx`** — ein strukturierter Prüfbericht (A4 quer) mit
Deckblatt, Inhaltsverzeichnis, 11 nummerierten Kapiteln, der Pflichtspalten-Nachweismatrix, den
beiden formatierten Testfall-Tabellen (alle Pflichtspalten) und dem **Anhang A mit den Screenshots
Abb. 1 (BUG-01) und Abb. 2 (BUG-02)** (Bilder direkt in die `.docx` eingebettet — keine Videodateien).

Falls eine **PDF** verlangt ist: das Word-Dokument öffnen und *Datei → Exportieren/Sichern als → PDF*.

Neu erzeugen (rendert Screenshots + baut die `.docx`):
```bash
python3 -m venv /tmp/a4venv && /tmp/a4venv/bin/pip install Pillow python-docx
/tmp/a4venv/bin/python build-docx.py
```
Die Screenshots zeigen den realen Konsolenmitschnitt (Request, HTTP-Status, Antwort — „Erwartet 400 vs. Ist 200").

> Die Screenshots wurden aus den echten Bug-Durchläufen gerendert (`bugs/BUG-0X.png`, Quelle:
> `bugs/BUG-0X.txt`). Falls dein Kurs einen „echten" Bildschirm-Screenshot verlangt: Backend starten,
> `bash bugs/BUG-01-future-end-zeit.sh` ausführen und mit `Cmd+Shift+4` abfotografieren.
