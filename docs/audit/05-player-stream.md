# Audit 05 — Player & Streaming End-to-End

**Source:** `internal/modules/player/handler.go`, `internal/modules/player/service.go`, `internal/modules/player/repository.go`, `internal/platform/storage/`, `internal/modules/player/routes.go`

---

## Endpoint Overview

| Endpoint | Auth | Purpose |
|----------|------|---------|
| `GET /api/v1/player/tracks/:id` | None (public) | Returns playback metadata (title, artist, album, cover, duration, **stream URL**) |
| `GET /api/v1/player/tracks/:id/stream` | None (public) | **Streams audio file directly** (proxies from storage) |
| `GET /api/v1/player/tracks/:id/cover` | None (public) | Proxies cover image |
| `GET /api/v1/admin/player/tracks/:id` | Bearer (AuthMW) | Admin: metadata including private tracks |
| `GET /api/v1/admin/player/tracks/:id/stream` | Bearer (AuthMW) | Admin: streams private tracks |

---

## Authentication & Authorization

### Public Endpoints (`/player/tracks/*`)
| Check | Implementation | Gap |
|-------|----------------|-----|
| **Track exists** | `repo.GetTrackForPlayback` → `ErrTrackNotFound` | ✅ |
| **Track is public** | `track.IsPublic` check in `getPlaybackTrack` (`service.go:69-72`) | ✅ Returns `ErrPrivateTrack` (403) |
| **Audio URL present** | `track.AudioURL != ""` check (`service.go:64-67`) | ✅ Returns `ErrAudioNotFound` (404) |
| **User auth** | **None** — no JWT check | ❌ **Anyone with track ID can stream** |
| **Premium/private check** | Only `IsPublic` flag; no premium tier logic | ⚠️ No subscription-gated content |

### Admin Endpoints (`/admin/player/tracks/*`)
| Check | Implementation | Gap |
|-------|----------------|-----|
| **Auth** | `AuthMW` (Bearer JWT) | ✅ |
| **Role** | No `AdminMW` on `/admin/player` routes — only `AuthMW` (`routes.go:16-22`) | ⚠️ Any authenticated user can access admin player endpoints |
| **Track visibility** | `GetAdminPlaybackTrack` skips `requirePublic` check (`service.go:48-50`) | ✅ Allows private tracks |

---

## Range Requests & HTTP 206/416 Support

### Current Implementation (`handler.go:98-126`, `128-164`)
```go
func (h *Handler) StreamTrack(c *gin.Context) {
    // ...
    stream, contentType, err := h.service.GetAudioContent(ctx, track.AudioURL)
    // ...
    c.Header("Content-Type", contentType)
    c.Header("Accept-Ranges", "bytes")        // SET but NOT HANDLED
    c.Header("Cache-Control", "public, max-age=3600, immutable")
    c.Header("Access-Control-Allow-Origin", "*")
    c.Header("Access-Control-Expose-Headers", "Accept-Ranges, Content-Range, Content-Length")
    c.Header("Cross-Origin-Resource-Policy", "cross-origin")
    io.Copy(c.Writer, stream)  // STREAMS ENTIRE FILE
}
```

### Missing Range Support
| Feature | Implemented? | Evidence |
|---------|--------------|----------|
| `Accept-Ranges: bytes` header | ✅ Set | `handler.go:119` |
| `Range` header parsing | ❌ **No** | No `c.Request.Header.Get("Range")` |
| `206 Partial Content` | ❌ **No** | Always returns 200 |
| `416 Range Not Satisfiable` | ❌ **No** | No validation |
| `Content-Range` header | ❌ **No** | Not set |
| `Content-Length` for partial | ❌ **No** | Full file length only |

### Impact
- **Seeking broken:** Mobile Safari, Chrome, Firefox cannot seek without Range support
- **Bandwidth waste:** Entire file sent even for short preview
- **Mobile playback issues:** iOS Safari may refuse to play without Range support

---

## Signed Stream URLs / Tokenized Access

### Current State
- **No signed URLs:** Stream URLs are predictable: `/api/v1/player/tracks/{id}/stream`
- **No token parameter:** No `?token=` or signed query string
- **No expiry:** URLs never expire
- **No user binding:** Anyone with track ID can stream

### Required for Private/Premium Content
| Requirement | Current | Needed |
|-------------|---------|--------|
| Short-lived signed URL (5-15 min) | ❌ | Generate via `storage.GetSignedURL(key, expiry)` |
| User-bound token | ❌ | Include `user_id` in signature |
| One-time use / play_id | ❌ | Include `play_id` for idempotency |
| Premium tier check | ❌ | Verify subscription in stream handler |

### Current Storage Interface (`internal/platform/storage/`)
- `GetURL(ctx, key) (string, error)` — returns direct URL (presigned or public)
- `GetObject(ctx, key) (io.ReadCloser, error)` — streams from S3/local
- **No `GetSignedURL` method** visible in interface

---

## Play Event Recording

### Current Flow (`player/service.go:269-317`)
```go
func (s *Service) TrackPlayed(ctx, userID, track, duration, completed, source) {
    _ = s.repo.IncrementPlayCount(ctx, track.ID)  // Hot-row UPDATE
    if s.publisher == nil { return }
    // Publishes TrackPlayedEvent to events bus
    // → analytics worker aggregates
}
```
- **Called from:** Not visible in handler — likely called from frontend via separate API call (`POST /library/history` or `/history/record`)
- **No server-side threshold:** Counts play on any request (even 1-second preview)
- **No server-side validation:** Trusts client `duration` and `completed` fields
- **Race:** `IncrementPlayCount` hot-row contention (see Audit 04)

---

## Redis Keys / Caches Involved

| Cache | Key Pattern | TTL | Invalidation |
|-------|-------------|-----|--------------|
| Track playback metadata | `player:track:{id}` | Not specified | Not specified |
| Popular tracks list | `catalog:popular` | Not specified | Not specified |
| Search results | `search:{query}:{page}` | Not specified | Not specified |
| Rate limits | `ratelimit:ip:...`, `ratelimit:user:...` | 1 minute | Auto-expiry |
| Auth sessions | `auth_sessions` table (PG) | 30 days (refresh) | `revoked_at` on logout |

**No Redis cache for audio streams** — streams directly from S3 via `storage.GetObject`.

---

## Storage Layer (`internal/platform/storage/`)

### Interface (from usage in `player/service.go`)
```go
type Storage interface {
    Exists(ctx context.Context, key string) (bool, error)
    GetURL(ctx context.Context, key string) (string, error)
    GetObject(ctx context.Context, key string) (io.ReadCloser, error)
    // Missing: GetSignedURL(ctx, key, expiry time.Duration) (string, error)
}
```

### Implementations
| Implementation | File | Notes |
|----------------|------|-------|
| S3 | `s3.go` | Uses AWS SDK v2; `GetURL` returns presigned URL if `S3_PRESIGN_URLS=true` else public base URL |
| Local | `local.go` | Filesystem; `GetURL` returns local path or error |

### `GetSignedURL` Missing
- **S3 implementation** supports presigned URLs via `S3_PRESIGN_URLS=true` but `GetURL` is used for metadata, not streaming
- **Streaming** uses `GetObject` (direct stream) not `GetURL`
- **No way** to generate short-lived signed stream URL for private content

---

## Headers Analysis (Stream Response)

| Header | Value | Correct? |
|--------|-------|----------|
| `Content-Type` | `audio/mpeg` (or detected) | ✅ |
| `Accept-Ranges` | `bytes` | ✅ Set but not honored |
| `Cache-Control` | `public, max-age=3600, immutable` | ✅ For public tracks |
| `Access-Control-Allow-Origin` | `*` | ✅ CORS for audio elements |
| `Access-Control-Expose-Headers` | `Accept-Ranges, Content-Range, Content-Length` | ✅ But not implemented |
| `Cross-Origin-Resource-Policy` | `cross-origin` | ✅ |
| `Content-Length` | Full file size | ❌ Should be partial for 206 |
| `Content-Range` | Missing | ❌ Required for 206 |
| `ETag` | Missing | ❌ Useful for caching |
| `Last-Modified` | Missing | ❌ |

---

## Security Assessment

| Risk | Severity | Evidence |
|------|----------|----------|
| **Public stream URLs** — anyone with track ID streams | High | `handler.go:98-126` no auth |
| **No Range support** — broken seeking, mobile issues | High | No `Range` parsing |
| **No signed URLs for private/premium** — predictability | High | `GetSignedURL` missing |
| **No play threshold** — 1-second play counts | Medium | Client-trusted `completed` |
| **No play_id idempotency** — retries double-count | Medium | `AddPlayHistory` no `play_id` |
| **Admin player endpoints lack AdminMW** | Low | `routes.go:16-22` only `AuthMW` |

---

## Recommendations

1. **Implement Range requests:** Parse `Range` header, return 206 with `Content-Range`, handle 416.
2. **Generate signed stream URLs:** Add `GetSignedURL` to storage interface; use for private/premium tracks.
3. **Add auth to public stream:** Require valid session or signed token; fallback to signed URL for `<audio>`.
4. **Server-side play threshold:** Only count play after N seconds (e.g., 30s) or `completed=true`.
5. **Add `play_id` idempotency** to play history endpoints.
6. **Fix admin player routes:** Add `AdminMW` middleware.
7. **Add `GetSignedURL` to storage interface** and implement in S3/local.

---

*Cite file:line from `internal/modules/player/handler.go:85-208`, `internal/modules/player/service.go:44-103`, `105-188`, `233-267`, `269-317`, `internal/modules/player/routes.go:7-23`, `internal/platform/storage/s3.go`, `local.go` (not fully read but referenced).*