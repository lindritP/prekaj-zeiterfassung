# CLAUDE.md — Prekaj-Zeiterfassung

> Dauerhafter Projektkontext für Claude Code. Diese Datei beschreibt **wie** gearbeitet wird
> (Stack, Struktur, Konventionen, Befehle). Die **granulare Aufgabenliste / der Umsetzungspfad**
> liegt in [`IMPLEMENTATION_PLAN.md`](./IMPLEMENTATION_PLAN.md) — dort wird Phase für Phase gearbeitet.

---

## 1. Schnellstart für Claude Code

- Lies **diese Datei** + [`IMPLEMENTATION_PLAN.md`](./IMPLEMENTATION_PLAN.md), bevor du Code schreibst.
- Arbeite **eine Phase / einen Block pro Session** ab (siehe Plan), nicht alles auf einmal.
- Vor "fertig": **`make check`** ausführen (lint + typecheck + build müssen grün sein).
- **Conventional Commits** (`feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`).
- **Niemals** Secrets, `.env`-Dateien oder Schlüssel committen (siehe `.gitignore`).
- Nach Änderungen an SQL-Queries/Migrations: **`make sqlc`** ausführen (Code neu generieren).
- Halte diese Datei aktuell, wenn sich Stack/Konventionen ändern; hake Aufgaben im Plan ab (`[x]`).
- Domänensprache ist **Deutsch** (siehe Glossar §5) — Entitäten/Felder konsistent so benennen.

---

## 2. Projektüberblick

Digitale Zeiterfassung & Urlaubsverwaltung für den Fliesenleger-Familienbetrieb **FliesenPrekaj GmbH**.
Löst Papier-/Zettelwirtschaft ab. Zwei Rollen:

- **Arbeiter** (Mitarbeiter): Zeiterfassung starten/beenden, eigene Zeiten & Überstunden einsehen,
  Urlaubsanträge stellen, eigene Dokumente/Lohnzettel herunterladen. → **Mobile-App**.
- **Admin** (Inhaber/Vater): Zeiten aller Arbeiter, Baustellen, Urlaubsanträge verwalten,
  Überstunden überwachen, Monatsberichte (PDF) erstellen. → **Web-App**.

**Sprachen:** UI & Fachsprache **Deutsch**. Code-Bezeichner, Kommentare, Commits, Docs: technisch
Englisch, aber **Fachbegriffe bleiben Deutsch** (`Arbeiter`, `Baustelle`, `Zeitbuchung`, …).

**Wichtige Rahmenbedingung:** DSGVO/GDPR — personenbezogene Daten, daher EU-Hosting,
Datensparsamkeit, Verschlüsselung, Auskunfts-/Löschfunktion (siehe §11).

---

## 3. Architektur-Entscheidungen (Decision Record)

| Thema | Entscheidung | Begründung |
|---|---|---|
| Backend | **Go 1.26**, REST-API, eigener Auth-Service | aus Projektvorgabe; performant, ein Binary |
| Datenbank | **PostgreSQL 17** | aus Projektvorgabe |
| Web-Frontend | **React + TypeScript + Vite + Tailwind CSS** | aus Projektvorgabe |
| Mobile-Frontend | **Expo SDK 56 (React Native 0.85) + TypeScript + Expo Router** | aktueller Expo-Stand 06/2026 |
| Mobile-Plattform | **iOS + Android** | bestätigt |
| iOS-Distribution | **Development Builds + EAS**, Apple Developer Account **vorhanden** | Expo Go im App Store hängt 06/2026 auf altem SDK fest → Dev Builds nötig |
| Repo | **Monorepo** (pnpm-Workspaces + Go-Modul), **public** | geteilte TS-Typen, ein CI-Setup; public bestätigt |
| Lokale Entwicklung | **macOS, MacBook M3 Pro (arm64)** | bestätigt |
| Hosting | **Azure Container Apps** (EU-Region) | bestätigt; serverlose Container, managed, günstig |
| DB-Hosting | **Azure Database for PostgreSQL – Flexible Server** (EU-Region) | managed, Backups/PITR, DSGVO-Region |
| Container | **Docker**, Multi-Stage; Images für **`linux/amd64`** | Container Apps läuft amd64 → auf M3 mit `buildx` cross-bauen |
| CI/CD | **GitHub Actions** (GitHub-hosted Runner) | bestätigt |
| Registry | **GitHub Container Registry (ghcr.io)** | public Repo → kostenlos, nah an Actions |
| Azure-Auth aus CI | **OIDC Federated Credentials** (keine Langzeit-Secrets) | Best Practice |
| IaC | **Bicep** | Azure-nativ |
| Testing | **Out of Scope** — separates Team / später | Vorgabe; Teststruktur wird nur vorbereitet, nicht ausgefüllt |

---

## 4. Monorepo-Struktur

```
prekaj-zeiterfassung/
├── CLAUDE.md                  # diese Datei
├── IMPLEMENTATION_PLAN.md     # granularer Umsetzungspfad (Checklisten)
├── README.md
├── Makefile                   # zentrale Befehle (siehe §6)
├── docker-compose.yml         # lokales Dev-Setup (db, api, web, adminer)
├── .editorconfig
├── .gitignore
├── .nvmrc                     # Node-Version (>= 20.19.4)
├── pnpm-workspace.yaml
│
├── backend/                   # Go-API + Auth + Geschäftslogik + PDF
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── config/            # Env-Konfiguration
│   │   ├── server/            # Router, Middleware, HTTP-Handler
│   │   ├── auth/              # JWT, Passwort-Hashing, Middleware
│   │   ├── domain/            # Arbeiter, Baustelle, Zeitbuchung, Urlaubsantrag, ...
│   │   ├── db/                # pgx-Pool, sqlc-generierter Code, Queries
│   │   ├── pdf/               # Monatsbericht / Lohnzettel
│   │   └── platform/          # Logging (slog), Fehler, Utils
│   ├── db/
│   │   ├── migrations/        # golang-migrate (*.up.sql / *.down.sql)
│   │   └── queries/           # *.sql für sqlc
│   ├── sqlc.yaml
│   ├── Dockerfile
│   ├── go.mod / go.sum
│   └── .env.example
│
├── web/                       # React + TS Admin-Oberfläche (Vite)
│   ├── src/
│   │   ├── api/               # typed API-Client + TanStack-Query-Hooks
│   │   ├── auth/              # Auth-Context, Token-Refresh, Guards
│   │   ├── components/
│   │   ├── pages/
│   │   └── lib/
│   ├── Dockerfile             # build + nginx (SPA-Fallback)
│   ├── nginx.conf
│   ├── index.html
│   ├── package.json
│   └── .env.example
│
├── mobile/                    # Expo / React Native Arbeiter-App
│   ├── app/                   # Expo Router (file-based)
│   ├── src/                   # api/, auth/, components/, lib/
│   ├── app.json / app.config.ts
│   ├── eas.json               # EAS-Build-Profile (development/preview/production)
│   ├── package.json
│   └── .env.example
│
├── packages/
│   └── shared/                # geteilte TS-Typen (DTOs), idealerweise aus OpenAPI generiert
│
├── infra/                     # Bicep-Module (Container Apps, PostgreSQL, Registry, ...)
│   └── main.bicep
│
└── .github/
    └── workflows/             # ci-backend.yml, ci-web.yml, ci-mobile.yml, deploy.yml
```

> **Mobile läuft NICHT in Docker** — Expo/RN-Entwicklung wird auf dem Mac direkt mit Expo CLI /
> Simulatoren gefahren; Release-Builds laufen über EAS.

---

## 5. Domänensprache (Ubiquitous Language)

Konvention: **Domänen-Entitäten & -Felder Deutsch (snake_case in DB, gleiche Nomen im Code),
generische Technik-Felder Englisch** (`id`, `created_at`, `updated_at`, `status`).

| Begriff | Bedeutung | Tabelle / Typ |
|---|---|---|
| Arbeiter | Mitarbeiter & zugleich Benutzer | `arbeiter` |
| Admin | Rolle Inhaber (in `arbeiter.rolle`) | `rolle = 'admin'` |
| Baustelle | Einsatzort | `baustelle` |
| Zeitbuchung | Eine Arbeitszeit-Erfassung (Start/Stopp) | `zeitbuchung` |
| Urlaubsantrag | Antrag auf Urlaub/Abwesenheit | `urlaubsantrag` |
| Überstunden / Saldo | Differenz Ist − Soll | berechnet, ggf. `ueberstunden_saldo` |
| Lohnzettel | Lohn-Dokument des Arbeiters (PDF) | `dokument` (Typ `lohnzettel`) |
| Monatsbericht | generiertes PDF pro Arbeiter/Monat | on-demand erzeugt |

**Zentrale Felder (Kern):**
- `zeitbuchung`: `arbeiter_id`, `baustelle_id`, `start_zeit`, `end_zeit` (NULL = läuft noch),
  `pause_minuten`, `notiz`.
- `urlaubsantrag`: `arbeiter_id`, `von_datum`, `bis_datum`, `typ`, `status`
  (`offen` | `genehmigt` | `abgelehnt`), `entschieden_von`, `entschieden_am`.
- `arbeiter`: `name`, `email`, `passwort_hash`, `rolle`, `wochenstunden`, `stundenlohn`, `aktiv`.

**Geschäftsregeln (Defaults — mit Auftraggeber bestätigen, siehe §13):**
- Pro Arbeiter darf **nur eine laufende** Zeitbuchung existieren (`end_zeit IS NULL`).
- Dauer = `end_zeit − start_zeit − pause_minuten`.
- Überstunden = Summe der Ist-Stunden im Monat − Monats-Soll (aus `wochenstunden` abgeleitet).

---

## 6. Befehle (Makefile)

> Alle Standardabläufe laufen über `make`. Lokale DB & Backend/Web via Docker Compose; Mobile via Expo CLI.

| Befehl | Zweck |
|---|---|
| `make bootstrap` | Abhängigkeiten installieren (Go-Module, pnpm install in web/mobile) |
| `make db-up` / `make db-down` | lokale PostgreSQL (Docker) starten/stoppen |
| `make migrate-up` / `make migrate-down` | DB-Migrationen anwenden/zurückrollen (golang-migrate) |
| `make migrate-new name=<name>` | neue Migration anlegen |
| `make sqlc` | type-safe DB-Code aus `db/queries/*.sql` generieren |
| `make run-api` | Backend lokal starten (`go run ./cmd/api`) |
| `make run-web` | Web-Frontend (Vite Dev-Server) starten |
| `make run-mobile` | Expo Dev-Server starten (`pnpm --dir mobile start`) |
| `make ios` / `make android` | Expo im iOS-Simulator / Android-Emulator starten |
| `make lint` | golangci-lint + eslint |
| `make typecheck` | `tsc --noEmit` (web + mobile) |
| `make build` | Backend-Binary + Web-Build erzeugen |
| `make check` | **lint + typecheck + build** (vor jedem Commit) |
| `make docker-build` | Docker-Images für `linux/amd64` bauen (buildx) |
| `make up` / `make down` | gesamtes lokales Stack via docker-compose |

> Falls ein Befehl noch nicht existiert: in der jeweiligen Phase im Plan anlegen (Phase 0/1).

---

## 7. Backend-Konventionen (Go)

- **Layout:** `cmd/api` = Einstieg; gesamte Logik in `internal/` (nicht importierbar von außen).
- **Router:** `chi` v5; Middleware-Stack: RequestID → Logger → Recoverer → CORS → RateLimit → Auth.
- **DB-Zugriff:** `pgx` v5 (`pgxpool`) + **`sqlc`** (kein ORM). SQL in `db/queries/`, generierter Code in `internal/db`.
- **Migrationen:** `golang-migrate`, Dateien in `db/migrations/` (`NNNN_name.up.sql` / `.down.sql`). Niemals angewandte Migration ändern — neue anlegen.
- **IDs:** UUID **v7** (zeitgeordnet) für API-sichtbare Entitäten (`github.com/google/uuid`).
- **Konfiguration:** ausschließlich über **Env-Variablen** (12-Factor). Lokal via `.env` (`godotenv`), Struct-Parsing via `caarlos0/env`. Keine Secrets im Code.
- **Logging:** stdlib **`log/slog`**, strukturiert (JSON in Prod). **Keine personenbezogenen Daten** loggen.
- **Fehler:** zentrales JSON-Fehlerformat `{ "error": { "code", "message" } }`; Handler geben saubere HTTP-Codes zurück.
- **Validierung:** `go-playground/validator` für Request-DTOs.
- **API-Versionierung:** Pfad-Präfix **`/api/v1`**. Health: `GET /healthz`, `GET /readyz`.
- **PDF:** `maroto` v2 (reines Go → keine externen Binaries im Container).
- **OpenAPI:** API-Spec pflegen (`backend/openapi.yaml`) und daraus die TS-Typen in `packages/shared` generieren (Single Source of Truth für Client-Typen).

---

## 8. Auth & Sicherheit

- **JWT, stateless.** Access-Token kurzlebig (~15 min) im `Authorization: Bearer`-Header für alle API-Calls.
- **Refresh-Token** langlebig + **rotierend**, serverseitig **gehasht** gespeichert (`refresh_token`-Tabelle) → Widerruf möglich.
  - **Web:** Refresh-Token in **httpOnly, Secure, SameSite=Strict-Cookie**; Access-Token nur im Speicher (JS). CSRF beachten.
  - **Mobile:** Refresh-Token in **`expo-secure-store`** (Keychain/Keystore).
- **Passwörter:** `bcrypt` (Cost ≥ 12) via `golang.org/x/crypto/bcrypt`.
- **Rollen:** Middleware prüft `rolle` (`arbeiter` / `admin`); Admin-Endpunkte sind getrennt.
- **Transport:** TLS überall (in Azure managed Zertifikat).
- **Härtung:** Rate-Limiting (`httprate`), Security-Header, strikte CORS-Allowlist (Web-Origin, App), Input-Validierung.
- **Seed:** ein initialer Admin (der Inhaber) per Seed/Migration anlegen.

---

## 9. Web-Frontend-Konventionen (React + TS)

- **Build:** Vite. **Styling:** Tailwind CSS (v4). Optional `shadcn/ui` für Komponenten.
- **Routing:** React Router; geschützte Routen via Auth-Guard.
- **Server-State:** **TanStack Query** v5 (Laden/Caching/Mutationen). Kein globaler Daten-Store nötig.
- **Formulare:** **React Hook Form + zod** (Schema-Validierung, shared mit DTO-Typen wo möglich).
- **API-Client:** zentraler Fetch-Wrapper mit Interceptor für automatischen Token-Refresh (401 → refresh → retry).
- **UX-Pflicht:** Lade-/Fehler-/Leerzustände, Toasts, optimistische Updates wo sinnvoll.
- **Lint:** ESLint + Prettier. **Typen** aus `packages/shared`.

---

## 10. Mobile-Konventionen (Expo SDK 56)

- **Expo Router** (file-based, `app/`), TypeScript, Expo SDK 56 / RN 0.85 / React 19.2. **Node ≥ 20.19.4**.
- **Dev-Workflow:** zuerst schnell mit **Expo Go im iOS-Simulator** (kostenlos, kein Apple-Account).
  Für **echte Geräte / native Module / Release** auf **Development Builds** wechseln (`expo-dev-client`,
  `npx expo run:ios` / `run:android`) — Expo Go im App Store hängt 06/2026 auf altem SDK fest.
- **Token-Storage:** `expo-secure-store`. **Server-State:** TanStack Query. **Formulare:** RHF + zod.
- **Builds/Distribution (EAS):** Profile `development`, `preview`, `production` in `eas.json`.
  - iOS → EAS Build → **TestFlight** (Apple Developer Account vorhanden) via `eas submit`.
  - Android → EAS Build (APK/AAB), interne Verteilung.
- **Kern-Screens:** Login, Zeiterfassung Start/Stopp (Timer), Meine Zeiten, Urlaubsantrag stellen,
  Meine Anträge, Dokumente/Lohnzettel. Offline-Robustheit ist **nicht** im Kern (später).

---

## 11. DSGVO / Datenschutz

- **EU-Region** für Container Apps **und** PostgreSQL (z. B. `germanywestcentral` / `westeurope` / `swedencentral`).
- TLS überall; Passwörter gehasht; **keine PII in Logs**.
- **Datensparsamkeit** — nur erfassen, was gebraucht wird.
- **Backups** verschlüsselt (Flexible Server PITR), definierte **Aufbewahrungsfrist**.
- **Betroffenenrechte:** Endpunkte/Funktionen für **Auskunft** (Datenexport) und **Löschung** eines Arbeiters vorsehen.
- **Secrets** via Container Apps Secrets / Azure Key Vault — nie im Repo.
- Mit Microsoft besteht ein **AVV/Data Processing Addendum** (Azure) — dokumentieren.

---

## 12. Container & Deployment (Kurzfassung)

- **Backend-Image:** Multi-Stage (`golang:1.26` Build → `gcr.io/distroless/static` o. ä.), statisches Binary, **`linux/amd64`**.
- **Web-Image:** Multi-Stage (Node-Build → `nginx:alpine`, SPA-Fallback auf `index.html`).
- **Lokal:** `docker-compose.yml` mit `postgres:17`, `api`, `web`, `adminer`.
- **Cloud:** Azure **Container Apps Environment** mit App `api` + App `web`; **PostgreSQL Flexible Server**;
  Images aus **ghcr.io**; Konfiguration/Secrets als Container-Apps-Secrets; Migrationen als **Container Apps Job**.
- **M3-Hinweis:** lokal mit `docker buildx --platform linux/amd64` bauen; in CI (Ubuntu-Runner = amd64) ohnehin nativ.

Details & Reihenfolge: siehe `IMPLEMENTATION_PLAN.md` Phasen 10–13.

---

## 13. Offene fachliche Entscheidungen (mit Auftraggeber klären)

Für die erste Implementierung sind unten **Defaults** angenommen; bitte mit dem Inhaber bestätigen,
bevor die jeweilige Logik final wird:

- **Überstunden-Bezug:** Tages-, Wochen- oder Monats-Soll? *(Default: Monats-Soll aus `wochenstunden`.)*
- Zählen **Urlaub / Krankheit / Feiertage** als Arbeitszeit für den Saldo? *(Default: Urlaub/Krank = Soll erfüllt, Feiertage später.)*
- **Minusstunden** führen? Rundung der Zeiten (z. B. auf 5/15 min)? *(Default: keine Rundung, Minusstunden erlaubt.)*
- **Pausen:** automatisch (gesetzlich) abziehen oder manuell erfassen? *(Default: manuell `pause_minuten`.)*
- **Lohnzettel:** vom Admin als PDF **hochgeladen** oder vom System **generiert**? *(Default: Upload durch Admin.)*
- **Urlaub:** Resturlaubs-Konto führen? Halbe Tage? Krankmeldung separat vom Urlaub? *(Default: nur Antrag/Status, ganze Tage.)*
- **Baustellenzuordnung** pro Zeitbuchung **Pflicht**? *(Default: optional.)*
- **Zeitzone / Sommerzeit**, Buchungen über Mitternacht, Überschneidungen. *(Default: Europe/Vienna, UTC speichern.)*
- Gibt es **mehrere Admins** oder nur den Inhaber? *(Default: ein Admin, Mehrere möglich.)*

---

## 14. Was bewusst NICHT Teil dieses Projekts ist

- **Tests** (Unit/Integration/E2E) — übernimmt separates Team / später. Es werden nur Teststruktur
  und ein grüner `go test ./...`-Schritt in CI vorbereitet, **keine** Tests ausgeschrieben.
- Anbindung externer Drittsysteme (alles in-App, inkl. Auth).
- Offline-First / komplexe Sync-Logik in der Mobile-App (frühestens nach dem Kern).
