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

- [ ] Migration `arbeiter`: `id` (uuid v7), `name`, `email` (unique), `passwort_hash`, `rolle` (`arbeiter`|`admin`), `wochenstunden`, `stundenlohn`, `aktiv`, `created_at`, `updated_at`.
- [ ] Migration `refresh_token`: `id`, `arbeiter_id` (FK), `token_hash`, `expires_at`, `revoked_at`, `created_at`.
- [ ] sqlc-Queries: User by email/id, create user, insert/get/revoke refresh-token.
- [ ] `internal/auth`: Passwort-Hashing (**bcrypt**, Cost ≥ 12).
- [ ] JWT (`golang-jwt/jwt/v5`): Access-Token (~15 min) + Refresh-Token (rotierend), Signatur via `JWT_SECRET`.
- [ ] Endpunkte: `POST /api/v1/auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/me`.
- [ ] **Web-Variante:** Refresh-Token als httpOnly/Secure/SameSite-Cookie setzen; CSRF berücksichtigen.
- [ ] **Mobile-Variante:** Refresh-Token im Body/Header (Client legt in SecureStore ab).
- [ ] Auth-Middleware: Token prüfen, `arbeiter`-Kontext setzen; **Rollen-Guard** (`requireAdmin`).
- [ ] **Seed**: initialen Admin (Inhaber) anlegen (Migration oder `make seed`).

**DoD:** Login liefert Tokens; geschützte Route nur mit gültigem Access-Token; Admin-Route blockt Arbeiter; Refresh rotiert.

---

## Phase 3 — Stammdaten: Arbeiter & Baustellen (Admin)

**Ziel:** Admin kann Mitarbeiter und Baustellen verwalten.

- [ ] Migration `baustelle`: `id`, `name`, `adresse`, `aktiv`, `created_at`, `updated_at`.
- [ ] sqlc-Queries: list/get/create/update/deactivate für `arbeiter` und `baustelle`.
- [ ] Endpunkte **Arbeiter (Admin):** `GET/POST /api/v1/arbeiter`, `GET/PATCH /arbeiter/{id}`, Deaktivieren (kein Hard-Delete).
- [ ] Endpunkte **Baustellen (Admin):** `GET/POST /api/v1/baustellen`, `GET/PATCH /baustellen/{id}`, Deaktivieren.
- [ ] Validierung der DTOs (`validator`); Eindeutigkeit E-Mail.

**DoD:** Admin legt Arbeiter & Baustelle an, bearbeitet und deaktiviert sie via API.

---

## Phase 4 — Zeiterfassung (Kernfunktion #1)

**Ziel:** Start/Stopp und Übersicht der Arbeitszeiten — funktioniert robust.

- [ ] Migration `zeitbuchung`: `id`, `arbeiter_id` (FK), `baustelle_id` (FK, nullable), `start_zeit`, `end_zeit` (nullable), `pause_minuten` (default 0), `notiz`, `created_at`, `updated_at`. Zeiten in **UTC**.
- [ ] DB-Constraint/Logik: **max. eine laufende** Buchung pro Arbeiter (`end_zeit IS NULL`) — Partial Unique Index.
- [ ] sqlc-Queries: start (insert), stop (update end_zeit), laufende holen, eigene listen (Zeitraum), admin-weite Liste (Filter: Arbeiter, Zeitraum, Baustelle).
- [ ] Endpunkte **Arbeiter:** `POST /api/v1/zeit/start`, `POST /zeit/stop`, `GET /zeit` (eigene, Zeitraum-Filter), `PATCH /zeit/{id}` (korrigieren), `GET /zeit/laufend`.
- [ ] Endpunkt **Admin:** `GET /api/v1/admin/zeit` (alle, Filter + Summen).
- [ ] Dauer-Berechnung (`end − start − pause`); Edge Cases: Stop ohne Start (409), Buchung über Mitternacht, negative Dauer verhindern.

**DoD:** Arbeiter startet/stoppt Zeiten; nur eine läuft gleichzeitig; eigene & (Admin) alle Zeiten mit korrekter Dauer abrufbar.

---

## Phase 5 — Urlaubsanträge

**Ziel:** Arbeiter stellt Anträge, Admin entscheidet.

- [ ] Migration `urlaubsantrag`: `id`, `arbeiter_id`, `von_datum`, `bis_datum`, `typ`, `status` (`offen`|`genehmigt`|`abgelehnt`), `grund`, `entschieden_von`, `entschieden_am`, `created_at`.
- [ ] sqlc-Queries: create, eigene listen, alle listen (Admin, Filter Status/Zeitraum), Status setzen.
- [ ] Endpunkte **Arbeiter:** `POST /api/v1/urlaub`, `GET /urlaub` (eigene), `DELETE /urlaub/{id}` (nur solange `offen`).
- [ ] Endpunkte **Admin:** `GET /api/v1/admin/urlaub`, `PATCH /admin/urlaub/{id}` (genehmigen/ablehnen, setzt `entschieden_von/am`).
- [ ] Validierung: `von_datum ≤ bis_datum`; Status-Übergänge nur erlaubt vom Status `offen`.

**DoD:** Antrag → Status `offen`; Admin genehmigt/lehnt ab; Arbeiter sieht Status seiner Anträge.

---

## Phase 6 — Überstunden-Logik

**Ziel:** transparenter Überstunden-Saldo (siehe offene Regeln in `CLAUDE.md` §13).

- [ ] `arbeiter.wochenstunden` als Soll-Basis nutzen; Monats-Soll daraus ableiten.
- [ ] Berechnung Saldo je Arbeiter/Monat: Ist (Summe `zeitbuchung`-Dauer) − Soll.
- [ ] Endpunkte: `GET /api/v1/ueberstunden` (eigene, je Monat) + `GET /admin/ueberstunden` (alle).
- [ ] Edge Cases entsprechend bestätigter Regeln (Urlaub/Krank/Feiertage, Rundung, Minusstunden).
- [ ] (Optional) Monatliche Persistenz `ueberstunden_saldo`, falls Performance/Historie nötig.

**DoD:** Saldo pro Arbeiter/Monat korrekt berechnet und über API abrufbar (eigene + Admin).

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
