# Muse Backend — AGENTS.md

Scope: this directory is the Go/Gin backend API (module `music`). Frontend (`frontend/`) and ML service (`moja-ml-service/`) are separate.

## Quick facts
- Language: Go 1.25.9, module `music`
- HTTP: Gin 1.12; entrypoint `cmd/api/main.go`
- DB: PostgreSQL 17; migrations in `migrations/` (Goose)
- Cache: Redis 8; Search: OpenSearch 3.2; Storage: local or S3
- Auth: JWT HS256; WebSocket: gorilla/websocket
- Logging: Zap; metrics: Prometheus `/metrics`; docs: Swagger at `/swagger/*`

## Commands (run from `backend/`)
- `make dev` / `make dev-worker` — infra + run
- `make test` — `go test ./... -race -count=1`
- `make lint-go` — golangci-lint; `make fmt` — gofmt
- `make build` / `make build-all`
- `make migrate-up` / `migrate-down` / `migrate-status`
- `make infra-up` / `infra-down` / `docker-up`

## Layout
- `cmd/api`, `cmd/worker`, `cmd/migrate` — entrypoints
- `internal/app` — bootstrap, DI container, routes, lifecycle
- `internal/common` — middleware, errors, pagination, response, validation
- `internal/config` — env/config loading & validation
- `internal/modules/<name>` — domain modules (model, repository, service, handler, dto, routes, tests)
- `internal/platform` — db, redis, opensearch, storage, ws, events, etc.
- `internal/workers` — async workers; import worker lives in `modules/importcmd/worker`
- `internal/pkg/audioinfo` — audio metadata helpers
- `migrations/` — Goose SQL migrations (000001..000064)
- `scripts/` — migrate.sh, dev helpers
- `docs/` — architecture, codebase, agent state, swagger

## Modules (internal/modules)
ai, analytics, auth, cacheadmin, catalog, contribution, covers, creator, dashboard, features, follow, gamification, health, history, importcmd, ingestion, library, lyrics, media, moderation, notification, permissions, player, playlist, queue, reactions, recommendation, reputation, search, social, subscription, upload, video.

## Where to find things
- Route registration: `internal/app/routes.go`
- DI wiring: `internal/app/container.go`, `bootstrap.go`
- Config: `internal/config/*`
- DB migrations: `migrations/*.sql`
- Events/outbox: `internal/platform/events`, `internal/modules/...` publishers
- Search normalization: `internal/modules/search` + helpers
- Media/storage: `internal/modules/media`, `internal/platform/storage`
- Auth/JWT: `internal/modules/auth`
- Swagger contract: `docs/swagger.json`, `docs/swagger.yaml`

## Deploy
- Docker: `Dockerfile` (ARG CMD_DIR=api|worker). Runtime needs ffmpeg, yt-dlp, node for ingestion. Healthcheck `/api/v1/health`.
- Fly.io: `fly.toml` (app music-api, region iad, internal_port 8080). Secrets: DATABASE_URL, REDIS_URL, JWT_SECRET, SENTRY_DSN, CORS_ALLOWED_ORIGINS, STORAGE_LOCAL_BASE_DIR.
- Railway/Render: see `docs/deploy/`. Vercel: frontend only; backend listens on PORT when provided.
- Compose: `docker-compose.yml` runs pg+redis+migrate+api+worker+frontend. Migrate service runs `cmd/migrate` once before api/worker.
- Migrations: never run inside HTTP request; `cmd/migrate` embeds `migrations/` FS. Production migrations via `scripts/migrate.sh` against Neon (human-only).
- Env: see `.env.example`. Key: APP_ENV, APP_HOST, APP_PORT/PORT, DATABASE_SOURCE/POSTGRES_*, REDIS_ADDR/PASSWORD, JWT_ACCESS_SECRET, JWT_REFRESH_SECRET, CORS_ALLOWED_ORIGINS, S3/MinIO keys, SENTRY_DSN, OTLP endpoint.

- Follow patterns in root `AGENTS.md` §5 (module shape, manual DI, events, response/errors).
- Never edit shipped migrations; add new ones.
- Run `make test` + `go vet` before committing.
- Persian text rules: NFKC, Arabic→Persian, strip ZWNJ/tatweel/diacritics, digit normalization, lowercase, whitespace collapse.
- Keep commits small (<~200 lines), Conventional Commits.

## Security audit (read-only, `docs/audit/`)

A full read-only backend security/concurrency audit was completed and written to `docs/audit/` (no source, tests, or migrations modified):

- `docs/audit/00-summary.md` — top 15 risks ranked by severity
- `docs/audit/01-routes.md` — full route inventory + swagger.json diff (~130 code routes undocumented, method/path mismatches)
- `docs/audit/02-middleware.md` — middleware stack, JWT validation, rate limits, CORS
- `docs/audit/03-schema.md` — 15 audited tables, columns, PK/FK/constraints/indexes
- `docs/audit/04-concurrency.md` — read-modify-write audit with races
- `docs/audit/05-player-stream.md` — player/stream end-to-end
- `docs/audit/06-tests.md` — per-package coverage, zero-test modules
- `docs/audit/07-duplicates-and-inconsistencies.md` — duplicates, envelopes, validation gaps

Top risks: (1) stream endpoint has no auth middleware (`player/routes.go:7-13`); (2) stateless access tokens, logout doesn't invalidate (`auth/service.go:138`); (3) `Access-Control-Allow-Origin: *` on media (`player/handler.go:121`); (4) no ingress body size limit; (5) possible duplicate active subscriptions on concurrent checkout; (6) two play-history tables (`listening_history` + `play_history`); (7) duplicate like/follow → 500 (no `ON CONFLICT`); (8) playlist position collisions → 500, no retry (`playlist/repository.go:325-470`); (9) play count not atomic with history/analytics; (10) refresh token revoke+create not atomic (`auth/service.go:124-135`); (11) swagger/code contract drift; (12) Range/206 advertised but not implemented (`player/handler.go:98-126`).
