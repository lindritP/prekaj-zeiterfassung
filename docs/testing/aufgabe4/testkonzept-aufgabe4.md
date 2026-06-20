# Aufgabe 4 — Testkonzept: spezifikationsbasierter Testfallentwurf

**Software Testing — Assignment 4** · **Autor:** Lindrit Prekaj · **Datum:** 2026-06-20 · **Gruppe:** 1 (solo)

Dieses Dokument beschreibt **Auswahl, Methodik und Herleitung** der Testfälle. Die fertige
Testfall-Tabelle inkl. Durchführungsergebnissen steht in [`testfaelle-aufgabe4.md`](./testfaelle-aufgabe4.md).

---

## 1. Anwendungsfall & Abgrenzung

Gewählter **Anwendungsfall (1 pro Gruppenmitglied → solo = 1): „Zeiterfassung starten / beenden"**
— der Arbeiter stempelt zu Arbeitsbeginn ein, zu Arbeitsende aus und korrigiert eine Buchung bei
Bedarf.

**Begründung der Auswahl:**
- **Höchstes Produktrisiko.** In Aufgabe 2 ist „Falsche Erfassung von Arbeitszeiten" mit Schaden
  *hoch* bewertet (fließt direkt in Lohnabrechnung/Finanzamt). Genau dieser Use Case ist betroffen.
- **Reichste spezifikationsbasierte Angriffsfläche.** Der Use Case besitzt einen echten
  **Zustandsautomaten** (keine → laufende → beendete Buchung), zahlreiche **Grenzwerte**
  (`end>start`, `pause≤Spanne`, `notiz≤1000`, Zukunfts-Start, gesetzliche Pause ab >6 h) und eine
  **Entscheidungslogik** für die Pausenberechnung — damit lassen sich alle gängigen
  Black-Box-Methoden sinnvoll anwenden.
- **Bewusste Abgrenzung zu Aufgabe 3.** Aufgabe 3 prüfte die *Überstunden-Berechnung* per
  **White-Box-Entscheidungsüberdeckung**. Aufgabe 4 prüft einen **anderen** Use Case rein
  **Black-Box/spezifikationsbasiert** → komplementäre Abdeckung.

**Keine User Story / kein Akzeptanzkriterien-Abgleich:** Die Anforderungen wurden in Aufgabe 1 als
**Anwendungsfälle (Use Cases)** beschrieben, nicht als User Stories. Der optionale zweite Absatz der
Aufgabenstellung („Sollten Sie User Stories … verwendet haben") entfällt daher; der Entwurf erfolgt
**anwendungsfallbasiert**.

---

## 2. Testbasis (Spezifikation)

Quelle: REST-API-Vertrag und Geschäftsregeln des Backends (`internal/server/zeitbuchung.go`,
Routen in `router.go`, Constraints in `db/migrations/000004_zeitbuchung.up.sql`).

**Endpunkte des Use Case** (alle unter `/api/v1`, JWT `Authorization: Bearer`):

| Methode | Pfad | Bedeutung |
|---|---|---|
| `POST` | `/zeit/start` | Buchung starten (201) |
| `POST` | `/zeit/stop` | laufende Buchung beenden (200) |
| `GET` | `/zeit/laufend` | aktuell laufende Buchung |
| `GET` | `/zeit/` | eigene Buchungen (Filter `?von=&bis=`) |
| `PATCH` | `/zeit/{id}` | Buchung korrigieren |
| `GET` | `/admin/zeit` | (nur Admin) alle Buchungen — für Rollen-Negativtest |

**Geschäftsregeln (Testbasis):**
1. Pro Arbeiter nur **eine** laufende Buchung (`end_zeit IS NULL`) → sonst 409.
2. `end_zeit` muss **echt nach** `start_zeit` liegen.
3. `pause_minuten` ≥ 0 **und** ≤ Spanne (`end−start` in Minuten).
4. `start_zeit` darf **nicht in der Zukunft** liegen.
5. `baustelle_id` ist **optional**, muss aber existieren, wenn angegeben (FK).
6. `notiz` ≤ **1000** Zeichen.
7. Zeiten werden **in UTC** gespeichert (Eingabe RFC3339, Offsets werden normalisiert).
8. **Gesetzliche Pause (§11 AZG):** Spanne **> 360 min ⇒ 30 min** Pause, sonst 0; `dauer = Spanne − pause` (≥ 0).
9. Buchungen sind **mandantengetrennt** (Arbeiter sieht/ändert nur eigene); `/admin/*` nur für Rolle `admin`.

---

## 3. Spezifikationsbasierte Entwurfsmethoden

Pro Methode wird gezeigt, **wie** daraus konkrete Testfälle entstehen (IDs siehe Testfall-Tabelle).

### 3.1 Äquivalenzklassenbildung (ÄK)

| Eingabe / Parameter | gültige Klasse(n) | ungültige Klasse(n) | Testfälle |
|---|---|---|---|
| `baustelle_id` | leer/null · existierende UUID | nicht existierende UUID | TC-START-03 / -01 / -04 |
| Authentifizierung | gültiges Worker-Token · gültiges Admin-Token | kein Token | TC-AUTH-01/-03, alle |
| Rolle auf `/admin/zeit` | `admin` | `arbeiter` | TC-AUTH-02 / -03 |
| `{id}` bei PATCH | eigene, existierende | fremde · unbekannte · maliforme | TC-PATCH-01 / -05 / -06 / -07 |
| Zeitfilter `von` | gültiges RFC3339 | nicht parsbar | TC-LIST-03 / -04 |

### 3.2 Grenzwertanalyse (GW)

| Größe | unmittelbar gültig | Grenze | unmittelbar ungültig | Testfälle |
|---|---|---|---|---|
| `notiz`-Länge | 999 | **1000 / 1001** | 1001 | TC-START-06 / -07 |
| `end` vs `start` | end>start | **end==start** | end<start | TC-STOP-03 / -04 (+ -01 gültig) |
| `pause` vs Spanne | pause<Spanne | **pause==Spanne / ==Spanne+1** | pause>Spanne | TC-PATCH-02 / -03 |
| `pause` Untergrenze | 0 | **0 / −1** | −1 | TC-PATCH-04 |
| Pausenschwelle Spanne | <360 | **360 / 361** | – | TC-STOP-05 / -06 |
| `start_zeit` Zeitachse | Vergangenheit | jetzt | **Zukunft** | TC-START-05 |
| `end_zeit` Zeitachse | Vergangenheit | jetzt | **Zukunft** | TC-STOP-07 *(→ BUG-01)* |

### 3.3 Entscheidungstabelle (ET) — Pausen-/Dauerberechnung beim Beenden

| Regel | Bedingung: Spanne > 360 min? | Aktion: `pause_minuten` | Aktion: `dauer_minuten` | Testfall |
|---|---|---|---|---|
| R1 | nein (≤ 360) | 0 | = Spanne | TC-STOP-05 (360 → 0 / 360) |
| R2 | ja (> 360) | 30 | = Spanne − 30 | TC-STOP-06 (361 → 30 / 331) |

Ergänzende Entscheidungstabelle für die **Pausen-Validierung bei PATCH**:

| Regel | Buchung beendet? | pause ≤ Spanne? | pause ≥ 0? | erwartetes Ergebnis | Testfall |
|---|---|---|---|---|---|
| P1 | ja | ja | ja | 200 (übernommen) | TC-PATCH-02 |
| P2 | ja | nein | ja | 400 „Pause überschreitet…" | TC-PATCH-03 |
| P3 | ja | – | nein | 400 (Validierung min 0) | TC-PATCH-04 |
| P4 | **nein (laufend)** | n/a (keine Spanne) | ja | **400 erwartet** | TC-PATCH-09 *(→ BUG-02: Ist 200)* |

### 3.4 Zustandsübergangstest (ZÜ)

**Zustände einer Zeitbuchung je Arbeiter:**
`KEINE` (keine laufende Buchung) · `LAUFEND` (`end_zeit = null`) · `BEENDET` (`end_zeit` gesetzt).

```
        start (201)               stop (200)
KEINE ───────────────▶ LAUFEND ───────────────▶ BEENDET
  ▲  \                   │  ▲                       │
  │   \ stop (409)       │  │ patch (200, bleibt    │ patch (200, bleibt
  │    ╰─ verboten       │  ╰─ LAUFEND)             ╰─ BEENDET)
  │                      │
  │  start auf LAUFEND ──╯ (409 verboten)
  ╰── stop auf KEINE/BEENDET (409 verboten)
```

| Ausgangszustand | Ereignis | erlaubt? | Zielzustand / Antwort | Testfall |
|---|---|---|---|---|
| KEINE | start | ja | LAUFEND (201) | TC-START-01/-03 |
| LAUFEND | start | nein | 409 `running_exists` | TC-START-02 |
| LAUFEND | stop | ja | BEENDET (200) | TC-STOP-01 |
| KEINE | stop | nein | 409 `no_running` | TC-STOP-02 |
| LAUFEND | patch (notiz) | ja | LAUFEND (200) | (Teil v. TC-PATCH-09-Setup) |
| BEENDET | patch (notiz/pause) | ja | BEENDET (200) | TC-PATCH-01/-02 |
| LAUFEND | get `/laufend` | – | Objekt | TC-LIST-01 |
| KEINE | get `/laufend` | – | leer/null | TC-LIST-02 |

### 3.5 Anwendungsfalltest (AF)

- **Hauptpfad:** einstempeln → arbeiten → ausstempeln → Buchung erscheint mit korrekter Dauer
  (TC-START-01 → TC-STOP-01 → TC-LIST-03).
- **Alternativpfade:** ohne Baustelle (TC-START-03), nachträgliche Korrektur (TC-PATCH-01/-02).
- **Ausnahmepfade:** doppeltes Einstempeln (TC-START-02), Ausstempeln ohne laufende Buchung
  (TC-STOP-02), fehlende/zu späte Eingaben (TC-START-05, TC-STOP-03/-04), fehlende Rechte
  (TC-AUTH-01/-02).

---

## 4. Werkzeuge & Durchführung

| Zweck | Werkzeug | Begründung |
|---|---|---|
| Testfall-Ausführung (Black-Box, API) | **Postman-Collection + Newman 6.2.2** | in Aufgabe 2 als System-/API-Testwerkzeug geplant; Assertions je Testfall, reproduzierbar, exportierbarer Bericht |
| Laufzeitumgebung | **docker-compose** (PostgreSQL 17) + lokale Go-API | produktionsnah, vom Produktivbetrieb getrennt (DSGVO) |
| Testdaten | Setup-Ordner der Collection legt **Admin-Login, Baustelle, Worker A/B** mit eindeutigen E-Mails an | jeder Lauf reproduzierbar aus definiertem Ausgangszustand |
| Fehlernachweis | **Repro-Skripte** (`bugs/BUG-0X*.sh`) + **Screenshots** (`bugs/BUG-0X.png`, im PDF eingebettet) | Ein-Befehl-Reproduktion + Bildnachweis je gefundenem Fehler |

Ausführung: `newman run postman/zeiterfassung.postman_collection.json -e postman/local.postman_environment.json`
→ Bericht unter `durchfuehrungsbericht/` (JSON, HTML, Text).

---

## 5. Rückverfolgbarkeit zu den Testzielen aus Aufgabe 2

| Testziel (Aufgabe 2) | abgedeckt durch |
|---|---|
| (1) Beginn/Ende korrekt & persistent erfassen | TC-START-01/-08, TC-STOP-01, TC-LIST-01/-03 |
| (2) gesetzliche Pause (>6 h) korrekt abziehen | TC-STOP-05/-06, TC-PATCH-02/-03 |
| (3) Nutzer nur auf eigene Daten | TC-PATCH-05, TC-LIST-01/-03 |
| (4) kein Zugriff auf fremde Daten | TC-PATCH-05, TC-AUTH-02 |

---

## 6. Fazit

31 spezifikationsbasiert abgeleitete Testfälle decken alle fünf Black-Box-Methoden sowie das
Top-Produktrisiko ab. **29** Fälle bestätigen das spezifizierte Verhalten, **2** decken reale
Abweichungen auf (**BUG-01** Zukunfts-`end_zeit`, **BUG-02** Pause auf laufender Buchung) — siehe
Testfall-Tabelle und `bugs/`. Der ursprüngliche Verdacht, dass `PATCH` die Spanne nicht prüft
(TC-PATCH-08), wurde durch die Durchführung **widerlegt** — ein Beleg dafür, dass die Fälle real
ausgeführt und nicht aus der Spezifikation „abgeschrieben" wurden.
