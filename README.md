# Prekaj-Zeiterfassung

Digitale Zeiterfassung & Urlaubsverwaltung für die **FliesenPrekaj GmbH**.
Monorepo: Go-Backend, React-Web (Admin), Expo-Mobile (Arbeiter), shared TS-Typen, Bicep-Infra.

> Konventionen: [`CLAUDE.md`](./CLAUDE.md) · Umsetzungspfad: [`IMPLEMENTATION_PLAN.md`](./IMPLEMENTATION_PLAN.md)

## Voraussetzungen (macOS / M3)

| Tool | Version | Install |
|---|---|---|
| Go | 1.26 | `brew install go` |
| Node | **22 LTS** (`.nvmrc`) | `nvm install && nvm use` |
| pnpm | 11.x | `npm install -g pnpm@11.6.0` (corepack ist auf Node 26 nicht dabei) |
| Docker + Compose | aktuell | Docker Desktop |
| Dev-CLIs | — | `make tools` |

> **Wichtig:** Die per `go install` installierten CLIs (`golangci-lint`, `sqlc`, `migrate`)
> landen in `$(go env GOPATH)/bin`. Dieses Verzeichnis muss in der `PATH` stehen:
>
> ```bash
> echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
> ```

> Node ist hier auf **22** gepinnt (`.nvmrc`), weil Expo SDK 56 Node 26 noch nicht
> offiziell unterstützt. Vor JS-Arbeit immer `nvm use` ausführen.

## Schnellstart

```bash
git clone <repo-url> && cd prekaj-zeiterfassung
nvm use                 # Node 22 aus .nvmrc
make tools              # einmalig: pnpm + Go-CLIs installieren
make bootstrap          # Go-Module + pnpm install

# Env-Dateien anlegen
cp backend/.env.example backend/.env
cp web/.env.example     web/.env
cp mobile/.env.example  mobile/.env
```

## Häufige Befehle

```bash
make help        # alle Targets
make check       # lint + typecheck + build (vor jedem Commit)
make db-up       # lokale PostgreSQL (ab Phase 1)
make run-api     # Backend lokal (ab Phase 1)
```

## Struktur

Siehe [`CLAUDE.md` §4](./CLAUDE.md). Kurz:
`backend/` (Go-API) · `web/` (React-Admin) · `mobile/` (Expo-Arbeiter) ·
`packages/shared/` (TS-DTOs) · `infra/` (Bicep) · `.github/workflows/` (CI/CD).

## Status

Aktuell **Phase 0** (Scaffold). Anwendungslogik folgt ab Phase 1 — siehe `IMPLEMENTATION_PLAN.md`.
