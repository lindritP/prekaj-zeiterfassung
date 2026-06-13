# Prekaj-Zeiterfassung — zentrale Befehle (CLAUDE.md §6)
# GNU Make 3.81 kompatibel (macOS default). Nur :=, ?=, $(shell), .PHONY.

SHELL := /bin/bash

# ── Tool-Versionen (pinned) ───────────────────────────────────
PNPM_VERSION     := 11.6.0
GOLANGCI_VERSION := v2.12.2
SQLC_VERSION     := v1.31.1
MIGRATE_VERSION  := v4.19.1

# ── Pfade ─────────────────────────────────────────────────────
GOBIN        := $(shell go env GOPATH)/bin
GOLANGCI     := $(GOBIN)/golangci-lint
SQLC         := $(GOBIN)/sqlc
MIGRATE      := $(GOBIN)/migrate

BACKEND_DIR  := backend
MIGRATIONS   := $(BACKEND_DIR)/db/migrations
DB_URL       ?= postgres://prekaj:prekaj@localhost:5432/prekaj?sslmode=disable

.DEFAULT_GOAL := help

# ── Meta ──────────────────────────────────────────────────────
.PHONY: help
help: ## Diese Hilfe anzeigen
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# ── Setup ─────────────────────────────────────────────────────
.PHONY: bootstrap
bootstrap: ## Abhängigkeiten installieren (Go-Module + pnpm install)
	@echo ">> Go-Module laden ..."
	cd $(BACKEND_DIR) && go mod download
	@echo ">> pnpm-Workspaces installieren ..."
	pnpm install -r
	@echo ">> bootstrap fertig. (Dev-CLIs separat: 'make tools')"

.PHONY: tools
tools: ## Dev-CLIs installieren (pnpm, golangci-lint, sqlc, migrate)
	@command -v pnpm >/dev/null 2>&1 || npm install -g pnpm@$(PNPM_VERSION)
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)
	@echo ">> Tools installiert nach $(GOBIN)"
	@echo ">> WICHTIG: $(GOBIN) muss in PATH sein (siehe README)."

# ── Datenbank (Docker) ────────────────────────────────────────
.PHONY: db-up
db-up: ## Lokale PostgreSQL starten (Docker) + auf Healthcheck warten
	docker compose up -d db
	@echo ">> Warte auf PostgreSQL (healthcheck) ..."
	@until [ "$$(docker inspect -f '{{.State.Health.Status}}' prekaj-db 2>/dev/null)" = "healthy" ]; do \
		printf '.'; sleep 1; \
	done; echo " bereit."

.PHONY: db-down
db-down: ## Lokale PostgreSQL stoppen
	docker compose stop db

# ── Migrationen (golang-migrate) ──────────────────────────────
.PHONY: migrate-up
migrate-up: ## Migrationen anwenden (Phase 1)
	@if [ -x "$(MIGRATE)" ]; then $(MIGRATE) -path $(MIGRATIONS) -database "$(DB_URL)" up; \
	else echo "[TODO Phase 1] migrate fehlt — 'make tools' & Migrationen anlegen"; fi

.PHONY: migrate-down
migrate-down: ## Letzte Migration zurückrollen (Phase 1)
	@if [ -x "$(MIGRATE)" ]; then $(MIGRATE) -path $(MIGRATIONS) -database "$(DB_URL)" down 1; \
	else echo "[TODO Phase 1] migrate fehlt"; fi

.PHONY: migrate-new
migrate-new: ## Neue Migration anlegen: make migrate-new name=<name>
	@if [ -z "$(name)" ]; then echo "Fehler: name=<name> angeben"; exit 1; fi
	@if [ -x "$(MIGRATE)" ]; then $(MIGRATE) create -ext sql -dir $(MIGRATIONS) -seq $(name); \
	else echo "[TODO Phase 1] migrate fehlt — 'make tools'"; fi

# ── Codegen ───────────────────────────────────────────────────
.PHONY: sqlc
sqlc: ## type-safe DB-Code generieren (Phase 1)
	@if [ -x "$(SQLC)" ] && [ -f "$(BACKEND_DIR)/sqlc.yaml" ]; then cd $(BACKEND_DIR) && $(SQLC) generate; \
	else echo "[TODO Phase 1] sqlc.yaml/Queries fehlen — 'make tools'"; fi

# ── Lokales Ausführen ─────────────────────────────────────────
.PHONY: run-api
run-api: ## Backend lokal starten (Phase 1)
	@if [ -f "$(BACKEND_DIR)/cmd/api/main.go" ]; then cd $(BACKEND_DIR) && go run ./cmd/api; \
	else echo "[TODO Phase 1] cmd/api/main.go fehlt noch"; fi

.PHONY: seed
seed: ## Initial-Admin anlegen/aktualisieren (Phase 2) — liest SEED_ADMIN_* aus .env
	@if [ -f "$(BACKEND_DIR)/cmd/seed/main.go" ]; then cd $(BACKEND_DIR) && go run ./cmd/seed; \
	else echo "[TODO Phase 2] cmd/seed/main.go fehlt"; fi

.PHONY: run-web
run-web: ## Web Dev-Server starten (Phase 8)
	@echo "[TODO Phase 8] pnpm --filter @prekaj/web dev"

.PHONY: run-mobile
run-mobile: ## Expo Dev-Server starten (Phase 9)
	@echo "[TODO Phase 9] pnpm --filter @prekaj/mobile start"

.PHONY: ios
ios: ## Expo im iOS-Simulator (Phase 9)
	@echo "[TODO Phase 9] pnpm --filter @prekaj/mobile ios"

.PHONY: android
android: ## Expo im Android-Emulator (Phase 9)
	@echo "[TODO Phase 9] pnpm --filter @prekaj/mobile android"

# ── Qualität ──────────────────────────────────────────────────
.PHONY: lint
lint: ## golangci-lint + eslint
	@if [ -x "$(GOLANGCI)" ]; then cd $(BACKEND_DIR) && $(GOLANGCI) run ./... || true; \
	else echo "[skip] golangci-lint fehlt — 'make tools'"; fi
	@command -v pnpm >/dev/null 2>&1 && pnpm -r --if-present run lint || echo "[skip] pnpm fehlt — 'make tools'"

.PHONY: typecheck
typecheck: ## tsc --noEmit (web + mobile)
	@command -v pnpm >/dev/null 2>&1 && pnpm -r --if-present run typecheck || echo "[skip] pnpm fehlt — 'make tools'"

.PHONY: build
build: ## Backend-Binary + Web-Build
	@if [ -f "$(BACKEND_DIR)/cmd/api/main.go" ]; then cd $(BACKEND_DIR) && go build -o bin/api ./cmd/api; \
	else echo "[skip] backend build — cmd/api fehlt (Phase 1)"; fi
	@command -v pnpm >/dev/null 2>&1 && pnpm -r --if-present run build || echo "[skip] pnpm fehlt — 'make tools'"

.PHONY: check
check: lint typecheck build ## lint + typecheck + build (vor jedem Commit)
	@echo ">> check OK"

# ── Container ─────────────────────────────────────────────────
.PHONY: docker-build
docker-build: ## Images für linux/amd64 bauen (Phase 10)
	@echo "[TODO Phase 10] docker buildx build --platform linux/amd64 ..."

.PHONY: up
up: ## Gesamtes Stack via docker-compose (Phase 10)
	@echo "[TODO Phase 10] docker compose up -d"

.PHONY: down
down: ## Stack stoppen (Phase 10)
	@echo "[TODO Phase 10] docker compose down"
