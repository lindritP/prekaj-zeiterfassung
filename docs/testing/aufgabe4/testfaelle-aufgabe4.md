# Aufgabe 4 — Testfälle & Durchführung: Anwendungsfall „Zeiterfassung starten/beenden"

**Software Testing — Assignment 4** · **Autor:** Lindrit Prekaj · **Gruppengröße:** 1 (solo → 1 Anwendungsfall)

Ausgewählter **Anwendungsfall:** *Zeiterfassung starten / beenden* (Arbeiter stempelt ein/aus,
inkl. Korrektur einer Buchung). Die Testfälle wurden mit **spezifikationsbasierten
Entwurfsmethoden** (Black-Box) abgeleitet — Herleitung siehe [`testkonzept-aufgabe4.md`](./testkonzept-aufgabe4.md).
Durchführung auf REST-API-Ebene mit **Postman-Collection + Newman** (das Web-Frontend hat nur
Login, die Mobile-App ist noch nicht implementiert → die API ist der zentrale testbare
Integrationspunkt).

> **Hinweis zur Tabellenform:** Aus Lesbarkeitsgründen ist die geforderte Tabelle in **Tabelle 1
> (Testfallentwurf)** und **Tabelle 2 (Testausführung)** aufgeteilt; beide sind über die
> **Testfall-ID** verknüpft. Tabelle 2 ist die geforderte *Erweiterung* der Entwurfstabelle um die
> Ausführungsattribute.

**Methoden-Legende:** **ÄK** = Äquivalenzklassenbildung · **GW** = Grenzwertanalyse ·
**ET** = Entscheidungstabelle · **ZÜ** = Zustandsübergangstest · **AF** = Anwendungsfalltest.

---

## Rahmendaten der Durchführung (gelten für alle Testfälle)

| Attribut | Wert |
|---|---|
| **Autor / Tester** | Lindrit Prekaj |
| **Datum der Durchführung** | 2026-06-20 |
| **Version der Software** | `0cae33d` (Branch `main`; Backend unverändert getestet) |
| **Umgebung** | „Lokal (Docker)" = macOS 26 (MacBook M3 Pro, arm64) · PostgreSQL 17 in Docker · Go-API `http://localhost:8080` · Ausführung über Postman-Collection mit **Newman 6.2.2** |
| **Testobjekt** | REST-API `/api/v1` des Backends (Endpunkte `/zeit/start`, `/zeit/stop`, `/zeit/laufend`, `/zeit/`, `/zeit/{id}`, `/admin/zeit`) |
| **Artefakte** | Collection: [`postman/zeiterfassung.postman_collection.json`](./postman/zeiterfassung.postman_collection.json) · Bericht: [`durchfuehrungsbericht/`](./durchfuehrungsbericht/) (Newman JSON/HTML/Text) |

**Gesamtergebnis:** 31 Testfälle · **29 erfolgreich** · **2 fehlerhaft** (TC-STOP-07 → BUG-01, TC-PATCH-09 → BUG-02) · 0 blockiert.
(Newman: 54 Requests inkl. Setup, 88 Assertions, 2 Fehlschläge — siehe Durchführungsbericht.)

---

## Tabelle 1 — Testfallentwurf

| ID | Titel | Methode | Getestete Anforderung (Anwendungsfall) | Vorbedingung | Benötigte Testdaten | Grober Testablauf | Erwartetes Ergebnis |
|---|---|---|---|---|---|---|---|
| TC-START-01 | Start Hauptpfad (Baustelle + Vergangenheit + Notiz) | AF, ÄK | Zeiterfassung starten (`POST /zeit/start`) | Worker A angemeldet, keine laufende Buchung; Baustelle B existiert | `baustelle_id`=B, `start_zeit`=2026-06-20T08:00:00Z, `notiz`="Fliesen Bad" | Start-Request senden | 201; `end_zeit`=null (läuft), `baustelle_id`=B |
| TC-START-02 | Start während laufender Buchung | ZÜ | Zeiterfassung starten (`POST /zeit/start`) | Worker A hat bereits eine laufende Buchung | leerer Body `{}` | erneut Start senden | 409 `running_exists` „Es läuft bereits eine Zeitbuchung." |
| TC-START-03 | Start ohne `baustelle_id` (optional) | ÄK | Zeiterfassung starten (`POST /zeit/start`) | keine laufende Buchung | `start_zeit`=2026-06-20T09:00:00Z, kein `baustelle_id` | Start senden | 201; `baustelle_id`=null |
| TC-START-04 | Start mit unbekannter `baustelle_id` | ÄK (ungültig) | Zeiterfassung starten (`POST /zeit/start`) | keine laufende Buchung | `baustelle_id`=`019ee601-0000-…-000` (nicht existent) | Start senden | 400 `bad_request` „Unbekannte Baustelle." |
| TC-START-05 | `start_zeit` in der Zukunft | GW (ungültig) | Zeiterfassung starten (`POST /zeit/start`) | keine laufende Buchung | `start_zeit`=2027-01-01T00:00:00Z | Start senden | 400 `bad_request` „…nicht in der Zukunft liegen." |
| TC-START-06 | `notiz` = 1000 Zeichen (Grenze gültig) | GW | Zeiterfassung starten (`POST /zeit/start`) | keine laufende Buchung | `notiz` = 1000×„x" | Start senden | 201 (obere gültige Grenze) |
| TC-START-07 | `notiz` = 1001 Zeichen (Grenze ungültig) | GW | Zeiterfassung starten (`POST /zeit/start`) | keine laufende Buchung | `notiz` = 1001×„x" | Start senden | 400 `validation_error` „notiz ist zu lang (max 1000)" |
| TC-START-08 | `start_zeit` mit `+02:00`-Offset | ÄK (Repräsentation) | Zeiterfassung starten (`POST /zeit/start`) | keine laufende Buchung | `start_zeit`=2026-06-19T10:00:00+02:00 | Start senden | 201; `start_zeit` als UTC `2026-06-19T08:00:00Z` |
| TC-STOP-01 | Stop Hauptpfad (Spanne 150 min) | AF, ZÜ | Zeiterfassung beenden (`POST /zeit/stop`) | laufende Buchung (Start 10:00) | `end_zeit`=2026-06-20T12:30:00Z | Stop senden | 200; `dauer_minuten`=150, `pause_minuten`=0 |
| TC-STOP-02 | Stop ohne laufende Buchung | ZÜ (ungültig) | Zeiterfassung beenden (`POST /zeit/stop`) | keine laufende Buchung | `{}` | Stop senden | 409 `no_running` „Keine laufende Buchung." |
| TC-STOP-03 | `end_zeit` == `start_zeit` | GW | Zeiterfassung beenden (`POST /zeit/stop`) | laufende Buchung (Start 10:00) | `end_zeit`=2026-06-20T10:00:00Z | Stop senden | 400 `bad_request` „end_zeit muss nach start_zeit liegen." |
| TC-STOP-04 | `end_zeit` < `start_zeit` | GW | Zeiterfassung beenden (`POST /zeit/stop`) | laufende Buchung (Start 10:00) | `end_zeit`=2026-06-20T09:00:00Z | Stop senden | 400 `bad_request` |
| TC-STOP-05 | Spanne **genau 360 min** | ET, GW | Pausenregel beim Beenden (`POST /zeit/stop`) | laufende Buchung (Start 08:00) | `end_zeit`=2026-06-20T14:00:00Z | Stop senden | 200; `pause_minuten`=0, `dauer_minuten`=360 (Grenze „>6 h") |
| TC-STOP-06 | Spanne **361 min** | ET, GW | Pausenregel beim Beenden (`POST /zeit/stop`) | laufende Buchung (Start 08:00) | `end_zeit`=2026-06-20T14:01:00Z | Stop senden | 200; `pause_minuten`=30, `dauer_minuten`=331 (gesetzl. Pause greift) |
| TC-STOP-07 | `end_zeit` in der **Zukunft** (2030) | GW (ungültig) | Zeiterfassung beenden (`POST /zeit/stop`) | laufende Buchung (Start 10:00) | `end_zeit`=2030-01-01T00:00:00Z | Stop senden | **400** — Arbeitszeit darf nicht in der Zukunft enden (analog `start_zeit`) |
| TC-PATCH-01 | Notiz korrigieren | AF | Buchung korrigieren (`PATCH /zeit/{id}`) | eigene **beendete** Buchung | `notiz`="korrigiert" | PATCH senden | 200; `notiz`="korrigiert" |
| TC-PATCH-02 | `pause` == Spanne (360) | GW | Buchung korrigieren (`PATCH /zeit/{id}`) | beendete Buchung, Spanne 360 | `pause_minuten`=360 | PATCH senden | 200; `pause_minuten`=360, `dauer_minuten`=0 (Grenze pause=Spanne) |
| TC-PATCH-03 | `pause` == Spanne+1 (361) | GW | Buchung korrigieren (`PATCH /zeit/{id}`) | beendete Buchung, Spanne 360 | `pause_minuten`=361 | PATCH senden | 400 `bad_request` „Pause überschreitet die Arbeitszeit." |
| TC-PATCH-04 | `pause_minuten` = -1 | GW | Buchung korrigieren (`PATCH /zeit/{id}`) | beendete Buchung | `pause_minuten`=-1 | PATCH senden | 400 `validation_error` (min 0) |
| TC-PATCH-05 | Fremde Buchung patchen (Ownership) | ÄK, Sicherheit | Buchung korrigieren (`PATCH /zeit/{id}`) | Worker B hat eine Buchung; Worker A angemeldet | `id` = Buchung von **B** | A patcht B's Buchung | 404 `not_found` (Mandanten-Trennung) |
| TC-PATCH-06 | Unbekannte `{id}` | ÄK (ungültig) | Buchung korrigieren (`PATCH /zeit/{id}`) | – | `id`=`019ee601-0000-…-000` | PATCH senden | 404 `not_found` |
| TC-PATCH-07 | Maliforme `{id}` | ÄK (ungültig) | Buchung korrigieren (`PATCH /zeit/{id}`) | – | `id`="not-a-uuid" | PATCH senden | 400 `bad_request` „Ungültige ID." |
| TC-PATCH-08 | `start_zeit` nach bestehendem `end_zeit` | GW (ungültig) | Buchung korrigieren (`PATCH /zeit/{id}`) | beendete Buchung (08:00–14:00) | `start_zeit`=2026-06-20T15:00:00Z | PATCH senden | 400 `bad_request` „end_zeit muss nach start_zeit liegen." |
| TC-PATCH-09 | `pause_minuten` auf **laufender** Buchung | ÄK, ZÜ | Buchung korrigieren (`PATCH /zeit/{id}`) | laufende (nicht beendete) Buchung | `pause_minuten`=30 | PATCH senden | **400** — Pause ohne Spanne nicht zulässig/prüfbar |
| TC-LIST-01 | `/laufend` bei laufender Buchung | ZÜ | Laufende Buchung abrufen (`GET /zeit/laufend`) | laufende Buchung (Start 13:00) | – | GET senden | 200; Objekt mit `id`, `end_zeit`=null |
| TC-LIST-02 | `/laufend` ohne laufende Buchung | ZÜ | Laufende Buchung abrufen (`GET /zeit/laufend`) | keine laufende Buchung | – | GET senden | 200; Body leer/`null` |
| TC-LIST-03 | Eigene Liste mit gültigem Zeitfilter | ÄK | Eigene Buchungen listen (`GET /zeit/`) | Worker A hat Buchungen | `von`=2026-06-01T00:00:00Z, `bis`=2026-07-01T00:00:00Z | GET senden | 200; JSON-Array |
| TC-LIST-04 | Eigene Liste mit ungültigem `von` | GW (ungültig) | Eigene Buchungen listen (`GET /zeit/`) | – | `von`=NOPE | GET senden | 400 `bad_request` „Ungültiger von-Zeitpunkt (RFC3339)." |
| TC-AUTH-01 | `/zeit/start` ohne Token | ÄK, Sicherheit | Authentifizierung | – | kein `Authorization`-Header | Start senden | 401 `unauthorized` „Kein gültiges Token." |
| TC-AUTH-02 | `/admin/zeit` mit Worker-Token | Sicherheit (Rolle) | Rollenbasierte Zugriffskontrolle | Worker A angemeldet | `Bearer` tokenA | GET `/admin/zeit` | 403 `forbidden` „Nur für Administratoren." |
| TC-AUTH-03 | `/admin/zeit` mit Admin-Token (Gegenprobe) | AF | Rollenbasierte Zugriffskontrolle | Admin angemeldet | `Bearer` adminToken | GET `/admin/zeit` | 200 |

---

## Tabelle 2 — Testausführung (Erweiterung, verknüpft über Testfall-ID)

> Datum = 2026-06-20 · Tester = L. Prekaj · Version = `0cae33d` · Umgebung = „Lokal (Docker)" (für **alle** Zeilen identisch, vgl. Rahmendaten).

| ID | Datum | Tester | Version | Umgebung | Ergebnis | Gefundene Abweichung(en) | Kommentare (konkret verwendete Testdaten) |
|---|---|---|---|---|---|---|---|
| TC-START-01 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 201; `end_zeit`=null, `baustelle_id` = angelegte Baustelle |
| TC-START-02 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 409 `running_exists` wie erwartet |
| TC-START-03 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 201; `baustelle_id`=null (optional bestätigt) |
| TC-START-04 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 400 „Unbekannte Baustelle." (FK-Prüfung) |
| TC-START-05 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 400; Meldung enthält „Zukunft" |
| TC-START-06 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 201 bei genau 1000 Zeichen |
| TC-START-07 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 400 `validation_error` bei 1001 Zeichen |
| TC-START-08 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | `+02:00` korrekt zu `08:00:00Z` normalisiert |
| TC-STOP-01 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 200; `dauer_minuten`=150, `pause_minuten`=0 |
| TC-STOP-02 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 409 `no_running` |
| TC-STOP-03 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 400 bei `end_zeit`=`start_zeit` (Grenze) |
| TC-STOP-04 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 400 bei `end_zeit`<`start_zeit` |
| TC-STOP-05 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | Spanne 360 → `pause`=0, `dauer`=360 (Grenze „>6 h") |
| TC-STOP-06 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | Spanne 361 → `pause`=30, `dauer`=331 |
| TC-STOP-07 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **fehlerhaft** | **BUG-01** → Screenshot **Abb. 1** ([`bugs/BUG-01.png`](./bugs/BUG-01.png)); Repro: [`bugs/BUG-01-future-end-zeit.sh`](./bugs/BUG-01-future-end-zeit.sh) | **Ist: HTTP 200** statt 400; `dauer_minuten`=**1.858.410** (≈1290 Tage). `end_zeit` wird gegen Zukunft **nicht** geprüft (bei `start_zeit` schon). Auch via `PATCH` reproduzierbar. |
| TC-PATCH-01 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 200; `notiz` aktualisiert |
| TC-PATCH-02 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | `pause`=360 → `dauer`=0 (Grenze pause=Spanne) |
| TC-PATCH-03 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 400 „Pause überschreitet die Arbeitszeit." |
| TC-PATCH-04 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 400 `validation_error` (min 0) |
| TC-PATCH-05 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 404 — A kann B's Buchung nicht ändern (Ownership ok) |
| TC-PATCH-06 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 404 `not_found` bei unbekannter, gültiger UUID |
| TC-PATCH-07 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 400 „Ungültige ID." bei „not-a-uuid" |
| TC-PATCH-08 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 400 — Spanne wird auch bei PATCH validiert (anfänglicher Verdacht widerlegt) |
| TC-PATCH-09 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **fehlerhaft** | **BUG-02** → Screenshot **Abb. 2** ([`bugs/BUG-02.png`](./bugs/BUG-02.png)); Repro: [`bugs/BUG-02-pause-auf-laufender-buchung.sh`](./bugs/BUG-02-pause-auf-laufender-buchung.sh) | **Ist: HTTP 200** statt 400; `pause_minuten`=30 auf laufender Buchung gesetzt, beim Stop still verworfen (Severity niedrig). |
| TC-LIST-01 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 200; Objekt mit `id`, `end_zeit`=null |
| TC-LIST-02 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 200; Body leer (Spezifikation nennt `null`) |
| TC-LIST-03 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 200; JSON-Array der eigenen Buchungen |
| TC-LIST-04 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 400 „Ungültiger von-Zeitpunkt (RFC3339)." |
| TC-AUTH-01 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 401 `unauthorized` ohne Token |
| TC-AUTH-02 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 403 `forbidden` — Worker auf Admin-Endpoint |
| TC-AUTH-03 | 2026-06-20 | L. Prekaj | 0cae33d | Lokal (Docker) | **Erfolgreich** | – | 200 — Admin-Gegenprobe |

---

## Gefundene Fehler (Zusammenfassung)

| Bug | Titel | Testfall | Schwere | Screenshot (Nachweis) | Repro-Skript |
|---|---|---|---|---|---|
| **BUG-01** | `end_zeit` in der Zukunft wird akzeptiert (Stop & PATCH) → absurde Dauer | TC-STOP-07 | mittel–hoch (direkt Arbeitszeit/Lohn → Top-Produktrisiko aus Aufgabe 2) | **Abb. 1** ([`bugs/BUG-01.png`](./bugs/BUG-01.png)) | [`bugs/BUG-01-future-end-zeit.sh`](./bugs/BUG-01-future-end-zeit.sh) · [Transcript](./bugs/BUG-01.txt) |
| **BUG-02** | `pause_minuten` auf laufender Buchung wird akzeptiert, beim Stop verworfen | TC-PATCH-09 | niedrig (irreführende Bestätigung, kein Datenverlust) | **Abb. 2** ([`bugs/BUG-02.png`](./bugs/BUG-02.png)) | [`bugs/BUG-02-pause-auf-laufender-buchung.sh`](./bugs/BUG-02-pause-auf-laufender-buchung.sh) · [Transcript](./bugs/BUG-02.txt) |

> Die Screenshots **Abb. 1** und **Abb. 2** (Konsolenmitschnitt des realen Bug-Durchlaufs mit
> Request, HTTP-Status und Antwort — „Erwartet 400 vs. Ist 200") sind im PDF unten im **Anhang**
> eingebettet und liegen als Bilddateien unter `bugs/BUG-0X.png`. Reproduktion und Verhalten sind
> zusätzlich textuell in `bugs/BUG-0X.txt` festgehalten.
