# Audit 04 — Concurrency & Race Conditions

**Source:** Service files, repositories, handlers; `internal/platform/events`, `internal/modules/*/service.go`, `internal/modules/*/repository.go`

---

## Read-Modify-Write Inventory

| Operation | File:Line | Transaction? | Locking / Constraint | Race Condition |
|-----------|-----------|--------------|---------------------|----------------|
| **Play count increment** | `player/service.go:281` `repo.IncrementPlayCount` | No (single UPDATE) | None (simple `UPDATE tracks SET play_count = play_count + 1`) | **Hot-row lock contention** on popular tracks; lost updates if not atomic (but single UPDATE is atomic in PG); high contention under load |
| **Play history insert** | `library/service.go:105-114` `AddPlayHistory` → `repo.AddPlayHistory` | No (single INSERT) | None | Client can send duplicate requests → double-count; no `play_id` idempotency key |
| **Queue add** | `queue/service.go` (needs check) | Likely single INSERT | Position via `SELECT MAX(position) + 1` | **Two concurrent adds read same max position** → duplicate position |
| **Queue reorder** | `queue/service.go` `ReorderTrack` | Likely single UPDATE | `SELECT ... FOR UPDATE` on playlist items? | **Two concurrent reorders** can swap incorrectly; need per-user advisory lock |
| **Playlist add track** | `playlist/service.go` `AddTrack` | Single INSERT | `uq_playlist_track` UNIQUE (`playlist_id`, `track_id`) | Duplicate insert → 409 (constraint catches it) ✅ |
| **Playlist reorder** | `playlist/service.go` `ReorderTrack` | Single UPDATE | `uq_playlist_position` UNIQUE (`playlist_id`, `position`) | **Concurrent reorders** can create gaps/duplicates; need transaction with `FOR UPDATE` |
| **Like track** | `library/service.go:27-55` `LikeTrack` | Single INSERT | `liked_tracks` PK (`user_id`, `track_id`) | Concurrent likes → PK violation → 409 (constraint) ✅ |
| **Follow artist** | `library/service.go:83-92` `FollowArtist` | Single INSERT | `followed_artists` PK (`user_id`, `artist_id`) | Concurrent follows → PK violation ✅ |
| **Subscription checkout** | `subscription/service.go` `Checkout` | Likely single INSERT | None visible | **Parallel checkouts** create multiple pending payments; no idempotency key |
| **Refresh token rotation** | `auth/service.go` `Refresh` | Single transaction | `auth_sessions` `refresh_token_hash` UNIQUE | **No reuse detection** — old refresh token not revoked on rotate; two concurrent refreshes with same token could both succeed |
| **Upload + track create** | `upload/service.go` `ReviewDraft` → `track.Create` | Multiple steps | None across steps | Partial failure leaves orphaned files/drafts |
| **Analytics event** | `analytics/service.go` `RecordEvent` | Single INSERT | None | Unbounded `metadata` JSON; anonymous endpoint → abuse/storage risk |

---

## Detailed Concurrency Analysis

### 1. Play Count Increment (`player/service.go:281`)
```go
_ = s.repo.IncrementPlayCount(ctx, track.ID)
```
- **Repository** likely: `UPDATE tracks SET play_count = play_count + 1 WHERE id = $1`
- **Race:** Single UPDATE is atomic in PostgreSQL, but **hot row** on viral track causes lock contention (row-level lock held until commit). Under high concurrency, throughput drops.
- **Fix:** Use Redis `INCR` with periodic flush to DB, or insert-only `play_events` table with `UNIQUE(user_id, play_id)` and async aggregation worker.

### 2. Play History Insert (`library/service.go:105-114`)
```go
func (s *Service) AddPlayHistory(ctx context.Context, input AddPlayHistoryInput) error {
    return s.repo.AddPlayHistory(ctx, input.UserID, input.TrackID, input.Duration, input.Completed)
}
```
- **No idempotency:** Client retries (network timeout) → duplicate row → inflated play count.
- **Fix:** Add `play_id` (client-generated UUID) to request; `UNIQUE(user_id, play_id)` constraint; `ON CONFLICT DO NOTHING`.

### 3. Queue Add / Reorder (`queue/service.go`)
- **Add:** `INSERT INTO queue_items (user_id, track_id, position) VALUES ($1, $2, (SELECT COALESCE(MAX(position), 0) + 1 FROM queue_items WHERE user_id = $1))`
  - **Race:** Two concurrent adds read same `MAX(position)` → same position assigned.
  - **Fix:** Use `pg_advisory_xact_lock(user_id_hash)` or `INSERT ... ON CONFLICT DO NOTHING` with generated position, then retry.
- **Reorder:** `UPDATE queue_items SET position = CASE ... WHERE user_id = $1`
  - **Race:** Two concurrent reorders can produce gaps/duplicates.
  - **Fix:** `SELECT ... FOR UPDATE` on affected rows in single transaction, or advisory lock.

### 3. Playlist Reorder (`playlist/service.go` `ReorderTrack`)
- Similar to queue: `UPDATE playlist_tracks SET position = CASE ... WHERE playlist_id = $1`
- **Constraint:** `uq_playlist_position` UNIQUE (`playlist_id`, `position`) helps but concurrent reorders can still deadlock or produce gaps.
- **Fix:** `SELECT ... FOR UPDATE` on all affected rows in single transaction.

### 4. Likes / Follows (Unique Constraint Protection)
- `liked_tracks` PK (`user_id`, `track_id`) → `INSERT ... ON CONFLICT DO NOTHING` pattern used in repo ✅
- `followed_artists` PK (`user_id`, `artist_id`) ✅
- **But:** Handler returns 409 on conflict (not idempotent PUT/DELETE) — client sees error instead of silent success.

### 5. Subscription Checkout (`subscription/service.go` `Checkout`)
- **No idempotency key:** Two clicks → two pending payments.
- **No callback/verify endpoint** for Zarinpal/IDPay — callback replay can charge twice.
- **Fix:** Require `Idempotency-Key` header; store with `UNIQUE` constraint; verify payment via provider callback with replay protection.

### 6. Refresh Token Rotation (`auth/service.go` `Refresh`)
```go
func (s *Service) Refresh(ctx context.Context, req RefreshRequest, ...) (AuthResponse, error) {
    // 1. Validate refresh token hash
    // 2. Revoke old session (set revoked_at)
    // 3. Create new session with new refresh token hash
    // 4. Generate new access + refresh tokens
}
```
- **Race:** Two simultaneous requests with same refresh token:
  - Both pass validation (token not yet revoked)
  - Both revoke old session (idempotent)
  - Both create new sessions → **two valid access tokens**
- **Fix:** Use `SELECT ... FOR UPDATE` on `auth_sessions` row during refresh, or `UPDATE ... SET revoked_at = NOW() WHERE refresh_token_hash = $1 AND revoked_at IS NULL` returning count; only one succeeds.

### 7. Cache Stampede
- **No `singleflight`** on cache misses for hot keys (track metadata, popular lists, recommendations).
- **Fix:** Use `singleflight.Group` in cacher (`internal/common/middleware/cacher.go`).

### 8. Multi-Device Streaming
- **No concurrency control** on simultaneous streams per account.
- WebSocket hub tracks per-user connections but no policy (allow 1, 2, unlimited?).
- **Fix:** Define policy; enforce via Redis counter on stream start/end.

### 9. Delete During Playback
- **Soft delete?** `tracks.is_public` flag; no `deleted_at` column.
- If track deleted (admin) while in queue/playlist/streaming: queue/playlist items become orphaned (FK CASCADE deletes them); streaming continues until buffer ends.
- **Fix:** Soft delete (`deleted_at`); check in stream handler; return 404/410.

---

## Locking / Constraint Summary

| Operation | Protection | Adequate? |
|-----------|------------|-----------|
| Play count | Single UPDATE (atomic) | ⚠️ Hot row contention |
| Play history | None (INSERT only) | ❌ No idempotency |
| Queue add | MAX(position) + 1 | ❌ Race on concurrent add |
| Queue reorder | None visible | ❌ Race on concurrent reorder |
| Playlist add | UNIQUE constraint | ✅ (409 on dup) |
| Playlist reorder | UNIQUE constraint | ⚠️ Gaps/duplicates possible |
| Like track | PK constraint | ✅ (409 on dup) |
| Follow artist | PK constraint | ✅ (409 on dup) |
| Subscription checkout | None | ❌ No idempotency |
| Refresh token | None (row lock missing) | ❌ Reuse possible |
| Upload + track | None across steps | ❌ Partial failure |

---

## Recommended Fixes Priority

1. **High:** Add idempotency keys to play history, subscription checkout, refresh token rotation
2. **High:** Fix play count hot row (Redis INCR + periodic flush)
3. **Medium:** Advisory locks for queue/playlist reorder
4. **Medium:** `singleflight` for cache misses
5. **Low:** PUT/DELETE idempotent likes/follows

---

*Cite file:line from `player/service.go:269-317`, `library/service.go:98-114`, `queue/service.go` (not fully read), `playlist/service.go`, `library/service.go:27-96`, `subscription/service.go`, `auth/service.go` (Refresh), `internal/common/middleware/cacher.go`.*