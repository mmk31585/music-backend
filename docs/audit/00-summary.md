# 00 — Audit Summary

Top 15 risks ranked by severity (security > data corruption > concurrency > contract > style). Each cites file:line.

## 1. Security — Stream endpoint has no auth middleware
Public audio for any public track. `GET /api/v1/player/tracks/:id/stream` and `/cover` are registered under `RegisterPublicRoutes` with **no AuthMW**.
- `internal/modules/player/routes.go:7-13`
- Impact: unauthenticated users can stream public audio.

## 2. Security — Access tokens are stateless; logout does not invalidate them
`Logout` revokes the refresh session only (`service.go:138`). Access-token JWTs have no blocklist/JTI revocation; a stolen access token is valid until expiry.
- `internal/modules/auth/service.go:138`, `jwt.go:87-113`
- Impact: stolen access tokens usable until TTL.

## 3. Security — `Access-Control-Allow-Origin: *` on media endpoints
Player stream/cover handlers set wildcard CORS, overriding the CORS middleware allowlist.
- `internal/modules/player/handler.go:121,159,205`
- Impact: media readable by any origin.

## 4. Security — No ingress body size limit
No `gin.MaxMultipartMemory` or `http.MaxBytesReader`. Media size limits are enforced only at service layer (200MB general).
- `cmd/api/main.go:82-91`, `container.go:360-364`
- Impact: large-body / DoS ingress.

## 5. Data corruption — Possible duplicate active subscriptions on concurrent checkout
No unique constraint on `(user_id, status='active')` and no `SELECT ... FOR UPDATE` on user subscription. Two concurrent checkouts could create two active subscriptions.
- `internal/modules/subscription/repository.go` (checkout path), `migrations/000006.sql:24-45`
- Impact: duplicate billing entitlement.

## 6. Data corruption — Two play-history tables, divergent truth
`listening_history` (`000010`, partitioned) and `play_history` (`000005`) store identical semantics. `history.RecordPlay` → `listening_history`; `library.AddPlayHistory` → `play_history`.
- `internal/modules/history/repository.go:110`, `library/repository.go:166`
- Impact: inconsistent history reads/recommendations.

## 7. Data corruption — Duplicate like/follow returns 500 (not idempotent)
`LikeTrack` (`INSERT`, no `ON CONFLICT`) and `FollowArtist` (`INSERT`, no `ON CONFLICT`) hit UNIQUE violation on concurrent duplicate → 500.
- `internal/modules/library/repository.go:58-70`, `follow/repository.go:51-68`
- Impact: client error on idempotent action.

## 8. Concurrency — Playlist position collisions → 500, no retry
`AddTrack`/`ReorderTrack` compute `max(position)+1` without `SELECT ... FOR UPDATE`; concurrent writers collide on `UNIQUE(playlist_id, position)`.
- `internal/modules/playlist/repository.go:325-470`
- Impact: 500 on concurrent playlist edits.

## 9. Concurrency — Play count increment not atomic with history/analytics
`IncrementPlayCount` is a single SQL UPDATE (atomic counter) but not coupled to history insert or `track_analytics` update; event/counter can diverge.
- `internal/modules/player/repository.go:66-68`, `service.go:269-317`
- Impact: play_count drift vs analytics.

## 10. Concurrency — Refresh token revoke+create not in one transaction
`RevokeSessionByID` then `CreateSession` are separate operations; crash between revoke and create leaves user without a session. Reuse detection via RowsAffected is correct, but non-atomic.
- `internal/modules/auth/service.go:124-135`
- Impact: lost session on crash.

## 11. Contract — Swagger/code mismatches; ~130 routes undocumented
Method mismatches (PATCH vs PUT on artists/albums/genres; POST vs PATCH on notifications), path mismatches (`/playlists` vs `/playlists/public`), wrong `@Router` prefix (`/api/v1/` doubled). ~130 code routes absent from `docs/swagger.json`.
- `internal/app/routes.go`, `docs/swagger.json`, handler godoc comments
- Impact: contract drift, generated clients wrong.

## 12. Contract — Range/206 advertised but not implemented
`Accept-Ranges: bytes` header set but no `Range` parsing; `io.Copy` streams full file with `200`. No `206`/`416`/`Content-Range`.
- `internal/modules/player/handler.go:98-126`, `stream.go:38-41`
- Impact: clients seeking audio receive full file.

## 13. Contract — Response envelope inconsistency
Some handlers return `{"success":bool,"data","message"}` (player), others `{"error":"..."}` (catalog 500), others `apperrors` envelope. No single standard.
- `internal/modules/player/handler.go:24-31`, `internal/common/response`
- Impact: clients must handle multiple shapes.

## 14. Concurrency — Enrich-all background job can run twice
`startPeriodicEnrichment` runs every 6h with no distributed lock; API + worker instances both run → duplicate enrichment events.
- `internal/app/routes.go:157-163`, `internal/modules/catalog/enrich.go:189`
- Impact: duplicate enrichment processing.

## 15. Style — RequestID middleware not wired into Gin router
`RequestID` is a stdlib `http.Handler` (`internal/common/middleware/request_id.go`) but `main.go` never calls `router.Use()` for it; `getRequestID()` always returns "".
- `cmd/api/main.go:83-91`, `internal/common/middleware/request_id.go:30`
- Impact: request IDs absent from logs/traces.

## Files created (read-only audit, no source changes)
- `docs/audit/00-summary.md` — this file
- `docs/audit/01-routes.md` — full route inventory + swagger diff
- `docs/audit/02-middleware.md` — middleware stack, JWT validation, rate limits
- `docs/audit/03-schema.md` — 15 audited tables, columns, PK/FK/constraints/indexes
- `docs/audit/04-concurrency.md` — RMW audit with races
- `docs/audit/05-player-stream.md` — player/stream end-to-end
- `docs/audit/06-tests.md` — per-package coverage, zero-test modules
- `docs/audit/07-duplicates-and-inconsistencies.md` — dups, envelopes, validation gaps

No source code, tests, or migrations were modified.
