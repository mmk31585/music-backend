# 06 — Tests

Result of `go test ./... -race -count=1 -cover` (run 2026-11-18, backend module `music`):

**All packages pass (no FAIL).** Zero failing or flaky tests detected in this run.

## Per-package coverage

| Package | Coverage | Has Tests? |
|---------|----------|------------|
| music/internal/app | — | Yes (app_test.go) |
| music/internal/common/middleware | — | Yes (cacher, cors, hmac_verify, ratelimit tests) |
| music/internal/config | — | Yes (config, features, storage) |
| music/internal/modules/ai | — | Yes (client, handler, repository, service) |
| music/internal/modules/auth | — | Yes (handler, repository, service) |
| music/internal/modules/catalog | — | Yes (enrich_test) |
| music/internal/modules/catalog/artist | 7.1% | Yes (repository_test) |
| music/internal/modules/catalog/track | 31.5% | Yes (repository_test) |
| music/internal/modules/history | 4.7% | Yes (signal_classifier_test) |
| music/internal/modules/ingestion | 11.0% | Yes (cleanup, enricher, mock, finalization, repository, service, testdata) |
| music/internal/modules/ingestion/enrichment | 22.2% | Yes |
| music/internal/modules/ingestion/finalization | 48.2% | Yes (service_test) |
| music/internal/modules/library | 2.5% | Yes (repository_test) |
| music/internal/modules/lyrics | 9.3% | Yes (handler_test) |
| music/internal/modules/media | 11.5% | Yes (handler, repository) |
| music/internal/modules/notification | 13.5% | Yes (repository_test) |
| music/internal/modules/playlist | 17.5% | Yes (handler, repository, service) |
| music/internal/modules/queue | 11.2% | Yes (repository_test) |
| music/internal/modules/recommendation | 32.1% | Yes (discover_weekly, home_recommendations, integration, radio, scorer, service, taste_profile) |
| music/internal/modules/search | 54.3% | Yes (persian, ranking, service_integration, service) |
| music/internal/modules/social | 28.1% | Yes (club, discussion, queue_engine, service, stage_manager) |
| music/internal/modules/subscription | 8.3% | Yes (repository_test) |
| music/internal/modules/upload | 31.8% | Yes (handler_test) |
| music/internal/modules/video | 13.7% | Yes (video_test) |
| music/internal/platform/storage | 13.2% | Yes (s3_test) |
| music/internal/platform/ws | 33.8% | Yes (hub_test) |

## Packages with ZERO tests

The following packages have no `*_test.go` files (coverage 0.0% or no test files):

- `music/internal/modules/catalog/album` (0.0%)
- `music/internal/modules/catalog/common` (0.0%)
- `music/internal/modules/catalog/genre` (0.0%)
- `music/internal/modules/catalog/stats` (0.0%)
- `music/internal/modules/contribution` (0.0%)
- `music/internal/modules/covers` (0.0%)
- `music/internal/modules/creator` (0.0%)
- `music/internal/modules/dashboard` (0.0%)
- `music/internal/modules/features` (0.0%)
- `music/internal/modules/follow` (0.0%)
- `music/internal/modules/gamification` (0.0%)
- `music/internal/modules/health` (0.0%)
- `music/internal/modules/importcmd` (+ acquisition, search, worker) (0.0%)
- `music/internal/modules/lyrics/lrc` (0.0%)
- `music/internal/modules/moderation` (0.0%)
- `music/internal/modules/permissions` (0.0%)
- `music/internal/modules/player` (0.0%) — **notably zero tests for the player/stream path**
- `music/internal/modules/reactions` (0.0%)
- `music/internal/modules/reputation` (0.0%)
- `music/internal/modules/importcmd/worker` (0.0%)
- `music/internal/pkg/audioinfo` (0.0%)
- `music/internal/platform/audio` (0.0%)
- `music/internal/platform/cache` (0.0%)
- `music/internal/platform/circuitbreaker` (0.0%)
- `music/internal/platform/database` (0.0%)
- `music/internal/platform/events` (0.0%)
- `music/internal/platform/fuzzy` (0.0%)
- `music/internal/platform/logger` (0.0%)
- `music/internal/platform/metrics` (0.0%)
- `music/internal/platform/opensearch` (0.0%)
- `music/internal/platform/payment` (0.0%)
- `music/internal/platform/sentry` (0.0%)
- `music/internal/platform/tracing` (0.0%)
- `music/internal/platform/web` (0.0%)
- `music/internal/workers/analytics` (0.0%)
- `music/internal/workers/cleanup` (0.0%)
- `music/internal/workers/recommendation` (0.0%)
- `music/migrations` — [no test files]

## Test characteristics

- Framework: `testify` (assert/require/suite), `miniredis` for Redis-dependent rate-limit/cache tests.
- Race detector enabled (`-race`) in CI per AGENTS.md §6.
- DB tests use mocks/sqlmock or real Postgres (per AGENTS.md) — several repository tests exist.
- No E2E/http-level test suite for route registration or middleware ordering.
- No tests for `StreamTrack`, `GetPlaybackTrack`, `IncrementPlayCount` concurrency.
