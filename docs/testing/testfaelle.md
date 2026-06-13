# Testfälle — Überstunden-Berechnung (`ueberstunden.go`)

**Testziel:** Entscheidungsüberdeckung (Zweigüberdeckung).
**Umsetzung:** `backend/internal/server/ueberstunden_test.go` (Go, table-driven).
**Wochentags-Anker:** 2023-01-01 = Sonntag, 2023-01-02 = Montag; Januar 2023 = 22 Werktage;
`monthBounds(2023,1)` = `[2023-01-01, 2023-02-01)`.

Status aller Testfälle: **PASS** (siehe `durchfuehrungsbericht.txt`).

---

## `monthBounds(jahr, monat)`

| ID | Eingabe | Erwartet | Abgedeckter Zweig |
|----|---------|----------|-------------------|
| MB1 | 2023, 1 | [2023-01-01, 2023-02-01) | Normalfall |
| MB2 | 2026, 12 | [2026-12-01, 2027-01-01) | Jahreswechsel Dez→Jan |

## `countWeekdays(start, end)` — Schleife + `wd != Sa && wd != So`

| ID | Eingabe | Erwartet | Abgedeckter Zweig |
|----|---------|----------|-------------------|
| CW1 | [01-01, 02-01) | 22 | Schleife durchläuft Monat, Werktage gezählt |
| CW2 | [Mo 01-02, Mo 01-09) | 5 | volle Woche, 5 Werktage |
| CW3 | [Sa 01-07, Mo 01-09) | 0 | Bedingung false → Wochenende übersprungen |
| CW4 | [01-02, 01-02) | 0 | Schleife 0× (start == end) |
| CW5 | [Mo 01-02, Di 01-03) | 1 | genau ein Werktag |

## `creditedWorkdays(rows, start, end)` — `from<start`-Clamp, `!d.After(to) && d.Before(end)`, Wochenend-Skip, Map-Dedup

| ID | Eingabe (rows) | Erwartet | Abgedeckter Zweig |
|----|----------------|----------|-------------------|
| CR1 | [Mo 01-02 … Fr 01-06] | 5 | `from<start` = false; Werktage gezählt |
| CR2 | [2022-12-28 … Di 01-03] | 2 | `from<start` = **true** → clamp auf start; So übersprungen |
| CR3 | [Mo 01-30 … 02-10] | 2 | `d.Before(end)` stoppt am Monatsende (end exklusiv) |
| CR4 | [Mo01-02…Mi01-04] + [Di01-03…Do01-05] | 4 | Überlappung → Map dedupliziert (nicht 3+3=6) |
| CR5 | [Sa 01-07 … So 01-08] | 0 | Wochenend-Bedingung false |
| CR6 | `nil` | 0 | äußere Schleife 0× |

## `sollMinuten(wochenstunden, tage)` — `ParseFloat`-Fehlerzweig

| ID | Eingabe | Erwartet | Abgedeckter Zweig |
|----|---------|----------|-------------------|
| SM1 | "40", 22 | 10560 | Parse ok |
| SM2 | "40.3", 1 | 484 | Parse ok + `math.Round` (483,6 → 484) |
| SM3 | "abc", 22 | 0 | `err != nil` = **true** → wh=0 |
| SM4 | "", 5 | 0 | `err != nil` = **true** (leerer String) |
| SM5 | "40", 0 | 0 | Parse ok, 0 Tage |

## `computeUeberstunden(...)` — `if effektiv < 0`

| ID | Eingabe | Erwartet | Abgedeckter Zweig |
|----|---------|----------|-------------------|
| CU1 | 2023-01, "40", ist=9000, Abw=5 Tage | Soll 8160, **Saldo +840**, Werktage 22, UrlaubKrank 5 | `effektiv<0` = false (Normalpfad, positives Saldo) |
| CU2 | 2023-01, "40", ist=0, keine Abw | Soll 10560, **Saldo −10560** | Normalpfad, negatives Saldo (Minusstunden) |

> **Infeasibler Zweig (dokumentierte Quality-Erkenntnis):** Der Zweig `effektiv < 0`
> (`effektiv = 0`) ist unerreichbar, weil stets `credited ≤ werktage` gilt — beide
> zählen Werktage im selben Fenster `[start, end)`. `go tool cover` zeigt daher
> `computeUeberstunden` mit 85,7 % (eine nicht erreichbare Anweisung). Empfehlung:
> als defensive Invariante kommentieren oder entfernen. Zweigüberdeckung der
> **erreichbaren** Pfade = 100 %.

## `parseMonat(w, r)` — `js=="" && ms==""`, `js=="" || ms==""`, jahr-Bereich, monat-Bereich

| ID | Query | Erwartet | Abgedeckter Zweig |
|----|-------|----------|-------------------|
| PM1 | (leer) | ok=true, jetzt (Jahr/Monat) | beide leer → Default `now` |
| PM2 | `jahr=2026` | 400 | genau einer fehlt (monat) |
| PM3 | `monat=6` | 400 | genau einer fehlt (jahr) |
| PM4 | `jahr=abc&monat=6` | 400 | `Atoi(jahr)` Fehler |
| PM5 | `jahr=1999&monat=6` | 400 | `jahr < 2000` |
| PM6 | `jahr=2101&monat=6` | 400 | `jahr > 2100` |
| PM7 | `jahr=2026&monat=xx` | 400 | `Atoi(monat)` Fehler |
| PM8 | `jahr=2026&monat=0` | 400 | `monat < 1` |
| PM9 | `jahr=2026&monat=13` | 400 | `monat > 12` |
| PM10 | `jahr=2026&monat=6` | ok=true, (2026,6) | gültiger Pfad |
| PM11 | `jahr=2000&monat=1` | ok=true, (2000,1) | untere Grenze gültig |
| PM12 | `jahr=2100&monat=12` | ok=true, (2100,12) | obere Grenze gültig |

---

**Summe: 24 Testfälle, alle PASS.** Coverage der reinen Funktionen (außer dem infeasiblen
Zweig) = 100 % → Entscheidungsüberdeckung erreicht.
