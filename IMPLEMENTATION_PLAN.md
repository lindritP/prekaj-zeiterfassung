# IMPLEMENTATION_PLAN.md — Prekaj-Zeiterfassung

> Granularer Umsetzungspfad. Arbeite **Phase für Phase** (und innerhalb einer Phase **Block für Block**)
> gemeinsam mit Claude Code ab. Hake erledigte Punkte mit `[x]` ab. Konventionen, Stack und Befehle
> stehen in [`CLAUDE.md`](./CLAUDE.md).

**Arbeitsregeln**
- Pro Claude-Code-Session möglichst **einen Block** umsetzen, dann `make check`, dann commit.
- Reihenfolge ist so gewählt, dass die **Core-Funktionalität** (Auth → Zeiterfassung) früh lauffähig ist.
- Jede Phase endet mit einer **Definition of Done (DoD)**.

---

## Phase 0 — Projekt-Setup & Tooling

**Ziel:** Monorepo steht, alle Tools laufen lokal auf dem Mac (M3).

- [x] Git-Repo lokal initialisiert (Branch `main`, erster Commit). **Public GitHub-Remote bewusst zurückgestellt** (vom Auftraggeber später anzulegen/pushen).
- [x] Monorepo-Ordnerstruktur gemäß `CLAUDE.md` §4 anlegen (`backend/`, `web/`, `mobile/`, `packages/shared/`, `infra/`, `.github/`).
- [x] `.gitignore` (Go, Node, Expo `ios/`+`android/`, `.env*`, Build-Artefakte, `*.tfstate`).
- [x] `.editorconfig` + `.nvmrc` (Node `22` — Expo SDK 56 unterstützt Node 26 noch nicht).
- [x] **pnpm** einrichten: `pnpm-workspace.yaml` (`web`, `mobile`, `packages/*`) + Root `package.json` (`pnpm@11.6.0` gepinnt).
- [x] Go geprüft (1.26.3), `backend/go.mod` initialisiert (`github.com/lindritP/prekaj-zeiterfassung/backend`).
- [x] `golangci-lint` (v2.12.2), `sqlc` (v1.31.1), `golang-migrate` (v4.19.1) via `make tools` installiert + im README dokumentiert.
- [x] **Makefile** mit allen Targets aus `CLAUDE.md` §6 (+ `tools`); ehrliche `[TODO Phase N]`-Platzhalter.
- [x] `.env.example` je App (`backend/`, `web/`, `mobile/`) + Root (`docker-compose`).
- [x] `README.md` mit Kurz-Setup ("So startest du lokal").

**DoD:** ✅ `make bootstrap` läuft fehlerfrei; `make check` exit 0; Repo-Struktur vollständig (16 `.gitkeep`). *Hinweis: Commits sind aktuell unsigniert committed (`--no-gpg-sign`), da der SSH-Signing-Key passphrasegeschützt ist — bei Bedarf nachsignieren.*

---

## Phase 1 — Backend-Grundgerüst & DB-Anbindung

**Ziel:** lauffähige Go-API mit DB-Verbindung, Migrations- und Codegen-Workflow.

- [x] `cmd/api/main.go`: Start, **Graceful Shutdown** (SIGINT/SIGTERM via `signal.NotifyContext`), Port aus Env.
- [x] `internal/config`: Env-Parsing (`caarlos0/env/v11`), lokal `godotenv`; Felder: `PORT`, `DATABASE_URL`, `JWT_SECRET`, `ENV`, `CORS_ORIGINS`, `LOG_LEVEL`.
- [x] `internal/platform`: **slog**-Logger (JSON in Prod, Text lokal) + Request-ID-Kontext; zentrales Fehler-Format `{ "error": { "code", "message" } }`.
- [x] `internal/server`: **chi v5**-Router, Middleware (RequestID → slog-Logger → Recoverer → CORS → RateLimit `httprate`). `RealIP` bewusst weggelassen (in v5.3.0 deprecated/spoofbar).
- [x] Healthchecks `GET /healthz` (liveness, immer 200) + `GET /readyz` (DB-Ping → 200/503).
- [x] `docker-compose.yml` (Root): `postgres:17` mit Volume + Healthcheck; `adminer` (Port 8081).
- [x] DB-Pool (`pgxpool`) in `internal/db/pool.go`, Verbindungsaufbau + Ping beim Start.
- [x] **golang-migrate** eingerichtet; erste Migration `000001_init` (Extension `pgcrypto`).
- [x] `make migrate-up/down/new` (bereits Phase 0 verdrahtet, jetzt genutzt).
- [x] **sqlc** (`sqlc.yaml`, pgx/v5-Driver) + Platzhalter-Query `health.sql`; `make sqlc` generiert sauber.
- [x] API-Versionierungs-Präfix `/api/v1` eingerichtet (gemountet, Routen folgen ab Phase 2).

**DoD:** ✅ `make db-up && make migrate-up && make run-api` startet; `/healthz` + `/readyz` liefern 200 (verifiziert: 503 bei DB-Stop, Recovery, Graceful Shutdown). `go build`/`go vet`/`golangci-lint` clean.

---

## Phase 2 — Authentifizierung & Benutzer (Core)

**Ziel:** Login/Logout/Refresh, rollenbasierter Zugriff, Seed-Admin.

- [x] Migration `arbeiter` (`000002_auth`): `id` (uuid v7, in Go erzeugt), `name`, `email` (citext UNIQUE), `passwort_hash`, `rolle` (TEXT+CHECK `arbeiter`|`admin`), `wochenstunden`/`stundenlohn` (numeric), `aktiv`, `created_at`, `updated_at`.
- [x] Migration `refresh_token`: `id`, `arbeiter_id` (FK, ON DELETE CASCADE), `token_hash` (bytea, sha256), `expires_at`, `revoked_at`, `created_at`.
- [x] sqlc-Queries: Arbeiter by email/id, create, UpsertAdmin; refresh-token create/get-by-hash/revoke/revoke-all. uuid→`google/uuid`, numeric→string, timestamptz→time.Time (sqlc-Overrides).
- [x] `internal/auth`: Passwort-Hashing (**bcrypt**, Cost ≥ 12, 72-Byte-Limit).
- [x] JWT (`golang-jwt/jwt/v5`): Access-Token (15 min, HS256, iss/sub/rolle/exp) + rotierender Refresh-Token (opaque, sha256-gehasht), Signatur via `JWT_SECRET`.
- [x] Endpunkte: `POST /api/v1/auth/login`, `/refresh`, `/logout`, `GET /me`.
- [x] **Web-Variante:** Refresh-Token als httpOnly/SameSite=Strict-Cookie (`Secure` nur in Prod — localhost ist http). CSRF: SameSite=Strict jetzt, Double-Submit-Token in Phase 13.
- [x] **Mobile-Variante:** Refresh-Token im JSON-Body (`client:"mobile"`); Client legt in SecureStore ab.
- [x] Auth-Middleware: `requireAuth` (Token prüfen, `Identity` in Kontext) + `requireAdmin`.
- [x] **Seed**: `cmd/seed` + `make seed` (idempotenter UpsertAdmin aus `SEED_ADMIN_*`).

**DoD:** ✅ Login liefert Tokens; `/me` nur mit gültigem Access-Token (sonst 401); `/admin/ping` blockt Arbeiter (403); Refresh rotiert + altes Token ungültig. `make check` clean. *(Temporäre `/admin/ping`-Probe — entfällt in Phase 3.)*

---

## Phase 3 — Stammdaten: Arbeiter & Baustellen (Admin)

**Ziel:** Admin kann Mitarbeiter und Baustellen verwalten.

- [x] Migration `baustelle` (`000003`): `id` (uuid v7), `name`, `adresse` (NOT NULL DEFAULT ''), `aktiv`, `created_at`, `updated_at`.
- [x] sqlc-Queries: list (aktiv-Filter)/get/create/update (COALESCE partial)/deactivate für `arbeiter` und `baustelle`.
- [x] Endpunkte **Arbeiter (Admin):** `GET/POST /api/v1/arbeiter`, `GET/PATCH/DELETE /arbeiter/{id}` (DELETE = soft-deactivate, kein Hard-Delete; Reaktivierung via `PATCH aktiv:true`). Passwort: Admin vergibt initiales `passwort`. Geld/Stunden als JSON-String.
- [x] Endpunkte **Baustellen (Admin):** `GET/POST /api/v1/baustellen`, `GET/PATCH/DELETE /baustellen/{id}`.
- [x] Validierung der DTOs (`go-playground/validator/v10`, JSON-Feldnamen in Fehlern); E-Mail-Eindeutigkeit (citext UNIQUE → 409). `passwort_hash` wird nie serialisiert.
- [x] Temporäre `/admin/ping`-Probe entfernt; Routen hinter `requireAuth + requireAdmin`.

**DoD:** ✅ Admin legt Arbeiter & Baustelle an, bearbeitet und deaktiviert sie via API. Verifiziert: 201/200, Liste + `?aktiv=`-Filter, 409 (dup E-Mail), 400 (Validierung/unknown field/oneof), 403 (Arbeiter), 404/400 (id), 401 (kein Token). `make check` clean. *(Offen für Phase 13: Last-Admin-Lockout-Schutz.)*

---

## Phase 4 — Zeiterfassung (Kernfunktion #1)

**Ziel:** Start/Stopp und Übersicht der Arbeitszeiten — funktioniert robust.

- [x] Migration `zeitbuchung` (`000004`): `id` (uuid v7), `arbeiter_id` (FK CASCADE), `baustelle_id` (FK SET NULL, nullable), `start_zeit`, `end_zeit` (nullable=läuft), `pause_minuten` (default 0, CHECK ≥0), `notiz`, timestamps. **UTC**. CHECK `end > start`.
- [x] **Partial Unique Index** `(arbeiter_id) WHERE end_zeit IS NULL` → max. eine laufende Buchung (Start #2 → 409).
- [x] sqlc-Queries: start/stop/laufend/get(own)/list(own, Zeitraum)/update(own); admin-Liste + Summen (gesamt + pro Arbeiter, in SQL). Dauer pro Zeile in Go.
- [x] Endpunkte **Arbeiter:** `POST /api/v1/zeit/start`, `/zeit/stop`, `GET /zeit` (Zeitraum), `PATCH /zeit/{id}`, `GET /zeit/laufend` — `requireAuth`, auf `identity.ArbeiterID` gescoped.
- [x] Endpunkt **Admin:** `GET /api/v1/admin/zeit` (Filter Arbeiter/Baustelle/Zeitraum + Summen).
- [x] Dauer = `end − start − pause`. **Pause = AUTO statutarisch (AZG §11: >6h → 30 min)**, override per PATCH. Edge Cases: Stop ohne Start → 409, über Mitternacht (UTC = unkritisch), negative Dauer verhindert (CHECK + `validateSpan`), Backdating erlaubt (Zukunft → 400), unbekannte Baustelle → 400.

**DoD:** ✅ Arbeiter startet/stoppt; nur eine läuft (409); eigene & Admin-Liste mit korrekter Dauer (inkl. auto-30min über 7h-Span) + Summen; Worker-Isolation (0/404). `make check` clean.

---

## Phase 5 — Urlaubsanträge

**Ziel:** Arbeiter stellt Anträge, Admin entscheidet.

- [x] Migration `urlaubsantrag` (`000005`): `id` (uuid v7), `arbeiter_id` (FK CASCADE), `von_datum`/`bis_datum` (date), `typ` (CHECK `urlaub`|`krankheit`|`sonstige`), `status` (CHECK `offen`|`genehmigt`|`abgelehnt`, default `offen`), `grund`, `entschieden_von` (FK), `entschieden_am`, `created_at`. CHECK `bis ≥ von`.
- [x] sqlc-Queries: create, eigene listen, get(own/admin), delete(own), admin-Liste (Filter Status/Zeitraum), decide (atomar `WHERE status='offen'`). sqlc `date`→`time.Time`-Override.
- [x] Endpunkte **Arbeiter:** `POST /api/v1/urlaub`, `GET /urlaub` (eigene), `DELETE /urlaub/{id}` (nur `offen` → sonst 409). `requireAuth`, gescoped.
- [x] Endpunkte **Admin:** `GET /api/v1/admin/urlaub` (Filter), `PATCH /admin/urlaub/{id}` (genehmigen/ablehnen → setzt `entschieden_von`=Admin + `entschieden_am`).
- [x] Validierung: `von_datum ≤ bis_datum` (app + DB-CHECK); Status-Übergänge nur aus `offen` (Re-Entscheidung → 409). Daten als `JJJJ-MM-TT` (validator `datetime`).

**DoD:** ✅ Antrag → `offen`; Admin genehmigt/lehnt ab (wer/wann erfasst); Arbeiter sieht Status. Verifiziert: 201/200, Filter, 409 (Löschen non-offen / Re-Entscheidung), 400 (von>bis/typ/datum/status), 404 (Isolation), 403 (Arbeiter→Admin-Route). `make check` clean. *(§13 offen: Resturlaub-Konto, halbe Tage, Überlappung.)*

---

## Phase 6 — Überstunden-Logik

**Ziel:** transparenter Überstunden-Saldo (siehe offene Regeln in `CLAUDE.md` §13).

- [x] `arbeiter.wochenstunden` als Soll-Basis; **Monats-Soll = (wochenstunden/5) × Werktage(Mo–Fr)** (workdays-basiert, bestätigt).
- [x] Berechnung Saldo je Arbeiter/Monat: Ist (Summe `zeitbuchung`-Dauer im Monat) − effektives Soll. On-demand (keine Tabelle).
- [x] Endpunkte: `GET /api/v1/ueberstunden` (eigene, `?jahr=&monat=`, Default aktueller Monat) + `GET /api/v1/admin/ueberstunden` (alle aktiven oder `?arbeiter=`).
- [x] Regeln (bestätigt): **Urlaub/Krankheit = Soll erfüllt** (genehmigte Tage reduzieren Soll), keine Rundung, **Minusstunden erlaubt**. Feiertage **noch nicht** ausgenommen (§13 "später").
- [ ] (Optional) Monatliche Persistenz `ueberstunden_saldo` — zurückgestellt (on-demand reicht aktuell).

**DoD:** ✅ Saldo pro Arbeiter/Monat korrekt (verifiziert: Soll 480×(22−5)=8160, Ist 480, Saldo −7680) — eigene + Admin (Array + `?arbeiter=`). Edges: monat=13/jahr-allein → 400, Arbeiter→Admin-Route → 403. `make check` clean.

---

## Phase 7 — PDF-Reporting & Dokumente

**Ziel:** Monatsbericht (Admin) generieren, Lohnzettel-Download (Arbeiter).

- [ ] `maroto` v2 in `internal/pdf` einrichten (Layout-Helfer).
- [ ] **Monatsbericht (Admin):** `GET /api/v1/admin/berichte/monat?arbeiter=&jahr=&monat=` → PDF (Zeiten, Summe, Überstunden, Baustellen).
- [ ] Migration `dokument`: `id`, `arbeiter_id`, `typ` (`lohnzettel`|…), `jahr`, `monat`, `pfad`/`blob_ref`, `created_at`.
- [ ] **Lohnzettel (Default: Upload durch Admin):** `POST /api/v1/admin/dokumente` (Upload), `GET /api/v1/dokumente` (Arbeiter: eigene), `GET /dokumente/{id}/download`.
- [ ] Datei-Storage: lokal Volume/Verzeichnis (Dev) — in Phase 12 auf **Azure Blob Storage** umstellen.
- [ ] Zugriffsschutz: Arbeiter sieht/lädt **nur eigene** Dokumente.

**DoD:** Admin erzeugt Monatsbericht-PDF; Arbeiter lädt eigenen Lohnzettel; Fremdzugriff blockiert.

---

## Phase 8 — Web-Frontend (Admin-Oberfläche)

**Ziel:** vollständige Admin-Web-App gegen die API.

- [ ] Vite + React + TS + Tailwind scaffolden; ESLint/Prettier; (optional) `shadcn/ui`.
- [ ] API-Client (Fetch-Wrapper) + **TanStack Query**; Auth-Interceptor (401 → refresh → retry).
- [ ] Auth-Context + **Login-Seite** + geschützte Routen (Guard); Logout.
- [ ] **Dashboard** (Kennzahlen-Überblick).
- [ ] **Arbeitszeiten (alle):** Tabelle + Filter (Arbeiter, Zeitraum, Baustelle) + Summen.
- [ ] **Arbeiter-Verwaltung:** Liste + Anlegen/Bearbeiten/Deaktivieren (RHF + zod).
- [ ] **Baustellen-Verwaltung:** Liste + CRUD.
- [ ] **Urlaubsanträge verwalten:** Liste + genehmigen/ablehnen.
- [ ] **Überstunden-Übersicht** je Arbeiter/Monat.
- [ ] **Monatsbericht:** Auswahl + PDF-Download; **Lohnzettel-Upload**.
- [ ] Lade-/Fehler-/Leerzustände, Toasts; Typen aus `packages/shared`.

**DoD:** Admin kann alle Admin-Use-Cases über die Web-App erledigen.

---

## Phase 9 — Mobile-App (Arbeiter)

**Ziel:** Arbeiter-App für die Kern-Use-Cases (iOS + Android).

- [ ] Expo (SDK 56) + Expo Router + TS initialisieren; `expo-dev-client`, `expo-secure-store`.
- [ ] API-Client + **TanStack Query**; Auth (Login, Token-Refresh, SecureStore).
- [ ] **Login-Screen.**
- [ ] **Zeiterfassung Start/Stopp** mit laufendem Timer + (optional) Baustellen-Auswahl + Notiz.
- [ ] **Meine Zeiten** (Liste/Filter, Tages-/Wochensummen).
- [ ] **Urlaubsantrag stellen** + **Meine Anträge** (Status).
- [ ] **Dokumente / Lohnzettel** (Liste + Download/Öffnen).
- [ ] **Überstunden** (eigener Saldo).
- [ ] Dev-Builds testen: `make ios` (Simulator) früh; dann `expo run:ios` / `run:android` auf echten Geräten.
- [ ] `eas.json`-Profile (`development`, `preview`, `production`) anlegen.

**DoD:** Arbeiter kann auf iOS-Simulator **und** Android Zeit erfassen, Urlaub beantragen, Dokumente laden.

---

## Phase 10 — Dockerisierung

**Ziel:** alle Dienste containerisiert, lokal per Compose lauffähig.

- [ ] **Backend-Dockerfile:** Multi-Stage (`golang:1.26` → `distroless/static`), statisches Binary, **`linux/amd64`**, non-root, Healthcheck.
- [ ] **Web-Dockerfile:** Multi-Stage (Node-Build → `nginx:alpine`); `nginx.conf` mit SPA-Fallback + Security-Headern.
- [ ] `.dockerignore` je App.
- [ ] `docker-compose.yml` erweitern: `db` + `api` + `web` (+ `adminer`), Netzwerke, Healthchecks, Env aus `.env`.
- [ ] **Migrations-Runner**: Image/Schritt, der `migrate-up` ausführt (für Compose & später Azure-Job).
- [ ] `make docker-build` (buildx, `--platform linux/amd64`) + `make up`/`make down`.

**DoD:** `make up` startet das komplette Stack lokal; Web erreicht die API; Migrationen laufen automatisch.

---

## Phase 11 — CI (GitHub Actions)

**Ziel:** automatische Qualitätssicherung bei jedem Push/PR; Images bauen.

- [ ] `ci-backend.yml`: `golangci-lint`, `go vet`, `go build`, **`go test ./...`** (grün halten — Tests folgen separat).
- [ ] `ci-web.yml`: `pnpm install`, `tsc --noEmit`, `eslint`, `vite build`.
- [ ] `ci-mobile.yml`: `pnpm install`, `tsc --noEmit`, `eslint` (EAS-Builds separat in Phase 12).
- [ ] Docker-Images bei Push auf `main`/Tags nach **ghcr.io** bauen & pushen (Tag = Git-SHA + `latest`).
- [ ] **Branch Protection** auf `main` (PR + grüne Checks erforderlich).
- [ ] **Dependabot** (Go-Module, npm, GitHub Actions); `CODEOWNERS`.
- [ ] Conventional-Commit-Check (optional) + PR-Template.

**DoD:** PRs werden automatisch geprüft; bei Merge auf `main` entstehen Images in ghcr.io.

---

## Phase 12 — CD & Azure-Deployment

**Ziel:** automatisches Deployment in Azure Container Apps (EU-Region).

- [ ] **Bicep (`infra/`):** Resource Group (EU-Region), **Container Apps Environment**, App `api`, App `web`,
      **PostgreSQL Flexible Server** (EU), Log Analytics, (optional) Key Vault, (optional) Azure Blob Storage für Dokumente.
- [ ] **GitHub → Azure OIDC**: App-Registration + **Federated Credentials** (keine Langzeit-Secrets); `azure/login`.
- [ ] Secrets/Config als **Container-Apps-Secrets** (`DATABASE_URL`, `JWT_SECRET`, …) bzw. Key Vault.
- [ ] `deploy.yml`: bei Tag/`main` → (1) Images aus ghcr.io referenzieren, (2) **Migrations als Container Apps Job** ausführen, (3) `az containerapp update` für `api` + `web`.
- [ ] **Custom Domain + managed TLS-Zertifikat** für die Web-App.
- [ ] Dokument-Storage von lokalem Volume auf **Azure Blob Storage** umstellen (aus Phase 7).
- [ ] **Mobile-Release (EAS):** `eas build` (iOS+Android) + `eas submit` iOS → **TestFlight**; Android interne Verteilung. `EXPO_TOKEN` als GitHub-Secret.

**DoD:** Push/Tag deployt API + Web nach Azure (EU); Migrationen laufen automatisch; Mobile-Builds via EAS erzeugbar.

---

## Phase 13 — DSGVO-Feinschliff & Härtung

**Ziel:** produktionsreif & datenschutzkonform.

- [ ] EU-Region für **alle** Dienste verifizieren.
- [ ] Security-Header (Web + API), strikte CORS-Allowlist, Rate-Limiting scharf stellen.
- [ ] **Auskunft** (Datenexport pro Arbeiter) + **Löschung**/Anonymisierung implementieren.
- [ ] Backup-/Restore-Strategie (Flexible Server PITR) testen; Aufbewahrungsfristen festlegen.
- [ ] Logging-Review: **keine PII** in Logs; sinnvolle Log-Level.
- [ ] Secrets-Review (nichts im Repo); Abhängigkeiten/CVEs prüfen (`govulncheck`, `pnpm audit`).
- [ ] Datenschutzhinweise + AVV (Azure) dokumentieren.

**DoD:** Sicherheits- & DSGVO-Checkliste vollständig; Restore einmal erfolgreich getestet.

---

## Spätere Erweiterungen (Backlog, nicht Kern)

- [ ] Offline-Erfassung + Sync in der Mobile-App.
- [ ] Push-Benachrichtigungen (z. B. Urlaub genehmigt).
- [ ] Feiertagskalender für Überstunden.
- [ ] Mehrsprachigkeit (i18n), falls nötig.
- [ ] Generierung statt Upload von Lohnzetteln (falls Auftraggeber das wünscht).
- [ ] **Testabdeckung** durch das separate Test-Team (Unit/Integration/E2E) — Struktur ist vorbereitet.
