# 07 — Duplicates and Inconsistencies

## Duplicate features

### 1. Play history — two tables, two endpoints

- `listening_history` table (`000010`, partitioned by `000043`) and `play_history` table (`000005`). Both store `(user_id, track_id, played_at)` plus duration/completed. Two sources of truth for the same concept.
- Endpoints: `POST/GET/DELETE /history/*` (`internal/modules/history/routes.go:6-14`) AND `POST/GET /library/history`, `GET /library/recently-played` (`internal/modules/library/routes.go:21-23`). Both record/list play history.
- `library/repository.go:166` `AddPlayHistory` inserts into `play_history`; `history/repository.go:110` `RecordPlay` inserts into `listening_history`. Inconsistent table choice for the same semantic action.

### 2. Follow — two tables, two endpoints

- `artist_follows` (`000009`) used by `internal/modules/follow/routes.go:6-7` (`POST/DELETE /artists/:id/follow`).
- `followed_artists` (`000005`) used by `internal/modules/library/routes.go:18-19` (`POST /library/artists/follow`) and `social/routes.go:30-31` (`POST /social/follow/:userId` — user follows, different table `user_follows`).
- Three follow concepts across `follow`, `library`, `social` modules with separate tables (`artist_follows`, `followed_artists`, `user_follows`).

### 3. Playlist tracks — three migration definitions, two active code paths

- `000004` (UUID PK, `position CHECK >0`, unique position), `000040` (BIGSERIAL, added_by), `000052` (BIGSERIAL, added_by). Only `000004` applies (`IF NOT EXISTS` no-ops for the others).
- `playlist/repository.go:325` `AddTrack` and `411` `ReorderTrack` use `000004` semantics.

### 4. Player routes duplicated

- `RegisterPrivateRoutes` and `RegisterAdminRoutes` in `internal/modules/player/routes.go:16-32` are identical (`/admin/player/tracks/:id` and `/stream`, AuthMW, same handlers). Registered: `RegisterPublicRoutes` (line 104) and `RegisterPrivateRoutes` (line 105). `RegisterAdminRoutes` is defined but not called in `routes.go` (dead code / duplicate).

### 5. Social follow duplicates library follow

- `social/routes.go:30-31` `POST/DELETE /social/follow/:userId` → `handler.Follow`/`Unfollow`.
- `library/routes.go:18-19` `POST/DELETE /library/artists/follow` → `handler.FollowArtist`/`UnfollowArtist` (artist follows).
- `follow/routes.go:6-11` `POST/DELETE /artists/:id/follow` and `/users/:id/follow`.

## Inconsistent response envelopes

- Player handler returns `{"success": bool, "data": any}` or `{"success": false, "message": string}` (`handler.go:24-31`).
- Auth handler returns `AuthResponse` struct (`dto.go`).
- Most handlers use `response.Wrap` / `response.Error` (`internal/common/response`).
- Errors: some return `{"error": "..."}` (e.g. catalog 500 observed `{"error":"failed to list tracks"}`), others `{"success":false,"message":"..."}`. No single envelope standard enforced.

## Inconsistent error formats

- `internal/common/errors` package (`apperrors`) with `CodeUnauthorized`, `CodeForbidden`, etc. used by auth middleware.
- Player handler uses plain `errors.Is` + `c.JSON` with ad-hoc `{"success":false,"message":...}` — does not use `apperrors`.
- Catalog returns `{"error":"failed to list tracks"}` on 500.

## Inconsistent pagination

- No standard pagination struct enforced across modules. `auth.AdminListUsers` uses `Page`/`PageSize` params; other list endpoints (catalog, playlist, social) use UNKNOWN pagination (not read in detail). `limit`/`offset` caps UNKNOWN for most list endpoints.

## Missing validation tags / no input limits

- `POST /analytics/events` (`analytics/routes.go:15`) — handler `TrackEvent` accepts `metadata JSONB` with no size cap; event_type CHECK-constrained at DB but no handler-level validation of payload size.
- Upload size: enforced at service layer (`MaxFileSizeBytes 200MB`, `container.go:360`) but **no ingress body size limit** (`gin.MaxMultipartMemory` not set, no `http.MaxBytesReader`) — see 02-middleware.
- `limit`/`offset` query params on list endpoints: UNKNOWN caps. No MAX_LIMIT constant found in common/pagination.
- `POST /library/history` and `POST /history/record` accept play events with no client-side validation of `ms_played`/`duration` bounds.

## Swagger vs code inconsistencies (from 01-routes)

- Method mismatches: PATCH vs PUT on `/artists/{artistID}`, `/albums/{albumID}`, `/genres/{genreID}`; POST vs PATCH on notifications read endpoints.
- Path mismatches: `/playlists` vs `/playlists/public`; `/library/tracks` vs `/library/tracks/liked`; `/library/recently-played` vs `/library/history/recent`.
- ~130 code routes have no swagger entry; all swagger paths have code counterparts (no swagger-only routes).
- Wrong `@Router` prefix: several swagger annotations include `/api/v1/` while `ginSwagger` is mounted under `/api/v1`, effectively doubling the prefix.
