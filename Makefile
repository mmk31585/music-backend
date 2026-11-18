GO           ?= go
SWAG         ?= $(shell go env GOPATH)/bin/swag
GOOSE        ?= $(shell go env GOPATH)/bin/goose
POSTGRES_URL ?= postgres://ava_user:ava_pass@127.0.0.1:5432/ava_db?sslmode=disable
PORT         ?= 8080

.PHONY: help swagger docs run api build migrate-up migrate-down migrate-status migrate dev tidy fmt lint test test-integration

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ── Swagger / OpenAPI ──────────────────────────────────────────────
swagger docs: ## Regenerate OpenAPI docs (swagger.json, swagger.yaml, docs.go) with swag
	@command -v $(SWAG) >/dev/null 2>&1 || { echo "swag not found: go install github.com/swaggo/swag/cmd/swag@latest"; exit 1; }
	$(SWAG) init -g cmd/api/main.go -o docs --parseDependency --parseInternal
	@echo "Swagger docs regenerated in docs/"

# ── Run API ─────────────────────────────────────────────────────────
run api: ## Run the API server (go run ./cmd/api)
	PORT=$(PORT) go run ./cmd/api

build: ## Build the API binary
	go build -o bin/api ./cmd/api

# ── Migrations ──────────────────────────────────────────────────────
migrate-up: ## Apply pending migrations (goose up)
	@command -v $(GOOSE) >/dev/null 2>&1 || { echo "goose not found: go install github.com/pressly/goose/v3/cmd/goose@latest"; exit 1; }
	POSTGRES_URL="$(POSTGRES_URL)" ./scripts/migrate.sh up

migrate-down: ## Roll back the last migration (goose down)
	POSTGRES_URL="$(POSTGRES_URL)" ./scripts/migrate.sh down

migrate-status: ## Show migration status
	POSTGRES_URL="$(POSTGRES_URL)" ./scripts/migrate.sh status

migrate: migrate-up ## Alias for migrate-up

# ── Dev stack ───────────────────────────────────────────────────────
dev: infra-up migrate run ## Start infra, run migrations, then run the API

# ── Helpers ─────────────────────────────────────────────────────────
tidy: ## go mod tidy
	go mod tidy

fmt: ## go fmt
	go fmt ./...

lint: ## golangci-lint
	golangci-lint run ./... --timeout 5m

test: ## go test
	go test ./... -race -count=1

test-integration: ## go test with integration tag
	go test -tags integration -count=1 ./internal/testutil/...

infra-up: ## Start postgres + redis via docker compose
	docker compose --env-file .env -f docker-compose.yml up -d postgres redis

infra-down: ## Stop infra
	docker compose --env-file .env -f docker-compose.yml down
