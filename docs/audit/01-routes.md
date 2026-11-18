# Audit 01 — Routes Inventory

**Source:** `internal/app/routes.go`, module `routes.go` files, `docs/swagger.json`  
**Base path:** `/api/v1` (from `api := r.Group("/api/v1")` in `routes.go:50`)

## Route Table

| Method | Full Path | Handler / Module | Middleware Chain | Required Role | In Swagger? | Notes |
|--------|-----------|------------------|------------------|---------------|-------------|-------|
| GET | `/swagger/*any` | swaggerFiles.Handler | — | — | N/A | Swagger UI |
| GET | `/metrics` | promhttp.Handler | — | — | N/A | Prometheus |
| GET | `/favicon.ico` | inline | — | — | N/A | 204 No Content |
| GET | `/health` | health.Handler | — | — | ✅ | Public |
| GET | `/features` | features.Handler | OptionalAuth, RateLimitOptional(60/min) | — | ✅ | |
| GET | `/dashboard/*` | dashboard.Handler | — | — | ✅ | |
| POST | `/auth/register` | auth.Handler.Register | — | — | ✅ | |
| POST | `/auth/login` | auth.Handler.Login | — | — | ✅ | |
| POST | `/auth/refresh` | auth.Handler.Refresh | — | — | ✅ | |
| POST | `/auth/logout` | auth.Handler.Logout | — | — | ✅ | |
| POST | `/auth/forgot-password` | auth.Handler.ForgotPassword | — | — | ✅ | |
| GET | `/auth/me` | auth.Handler.Me | AuthMW | user | ✅ | Bearer |
| GET/POST/PUT/DELETE | `/admin/users/*` | auth.Admin* | AuthMW, AdminMW | admin | ✅ | |
| GET/POST | `/permissions/*` | permissions.Handler | AuthMW, AdminMW | admin | ✅ | |
| GET/POST | `/reputation/*` | reputation.Handler | AuthMW | user | ✅ | |
| POST | `/uploads/drafts` | upload.Handler.CreateDraft | AuthMW | user | ✅ | |
| POST | `/uploads/drafts/:draftId/file` | upload.Handler.UploadDraftFile | AuthMW | user | ✅ | |
| GET | `/uploads/drafts/mine` | upload.Handler.ListMyDrafts | AuthMW | user | ✅ | |
| GET | `/uploads/drafts/:draftId` | upload.Handler.GetDraft | AuthMW | user | ✅ | |
| GET | `/uploads/slots` | upload.Handler.GetUploadSlots | AuthMW | user | ✅ | |
| GET | `/tracks/:trackId/co-uploaders` | upload.Handler.GetCoUploaders | AuthMW | user | ✅ | |
| POST/GET | `/uploads/review/*` | upload.Handler | AuthMW, RequireAnyPermission("review_uploads","manage_users") | moderator+ | ✅ | |
| POST/DELETE | `/uploads/co-uploaders/*` | upload.Handler | AuthMW, RequireAnyPermission("manage_users") | admin | ✅ | |
| GET/POST/PATCH/DELETE | `/admin/media/upload` | media.Handler | AuthMW, AdminMW | admin | ✅ | |
| GET/POST/PATCH/DELETE | `/admin/catalog/artist/*` | catalog.ArtistHandler | AuthMW, AdminMW | admin | ✅ | |
| GET/POST/PATCH/DELETE | `/admin/catalog/albums/*` | catalog.AlbumHandler | AuthMW, AdminMW | admin | ✅ | |
| GET/POST/PATCH/DELETE | `/admin/catalog/tracks/*` | catalog.TrackHandler | AuthMW, AdminMW | admin | ✅ | |
| GET/POST/PATCH/DELETE | `/admin/catalog/genres/*` | catalog.GenreHandler | AuthMW, AdminMW | admin | ✅ | |
| GET | `/admin/catalog/stats` | catalog.StatsHandler | AuthMW, AdminMW | admin | ✅ | |
| GET/POST/PATCH/DELETE | `/admin/ingestion/*` | ingestion.Handler | AuthMW, AdminMW | admin | ✅ | |
| GET | `/catalog/artists` | catalog.ArtistHandler.List | OptionalAuth, RateLimitOptional(60) | — | ✅ | |
| GET | `/catalog/artists/:artistID` | catalog.ArtistHandler.Get | OptionalAuth, RateLimitOptional(60) | — | ✅ | |
| GET | `/catalog/artists/:artistID/*` | various | OptionalAuth, RateLimitOptional(60) | — | ✅ | overview, tracks, albums, etc. |
| GET | `/catalog/albums` | catalog.AlbumHandler.List | OptionalAuth, RateLimitOptional(60) | — | ✅ | |
| GET | `/catalog/albums/:albumID` | catalog.AlbumHandler.Get | OptionalAuth, RateLimitOptional(60) | — | ✅ | |
| GET | `/catalog/albums/:albumID/*` | various | OptionalAuth, RateLimitOptional(60) | — | ✅ | |
| GET | `/catalog/tracks` | catalog.TrackHandler.ListPublic | OptionalAuth, RateLimitOptional(60) | — | ✅ | |
| GET | `/catalog/tracks/random` | catalog.TrackHandler.Random | OptionalAuth, RateLimitOptional(60) | — | ✅ | |
| GET | `/catalog/tracks/:trackID` | catalog.TrackHandler.Get | OptionalAuth, RateLimitOptional(60) | — | ✅ | |
| GET | `/catalog/tracks/:trackID/*` | various | OptionalAuth, RateLimitOptional(60) | — | ✅ | credits, artists |
| GET | `/catalog/genres` | catalog.GenreHandler.List | OptionalAuth, RateLimitOptional(60) | — | ✅ | |
| GET | `/catalog/genres/:genreID` | catalog.GenreHandler.Get | OptionalAuth, RateLimitOptional(60) | — | ✅ | |
| GET | `/lyrics/tracks/:trackId/lyrics` | lyrics.Handler.GetTrackLyrics | — | — | ⚠️ | Swagger shows `/api/v1/admin/lyrics` (doubled prefix) |
| GET | `/lyrics/tracks/:trackId/lyrics/all` | lyrics.Handler.GetLyricsByTrackID | — | — | ⚠️ | Same issue |
| POST/PUT/DELETE | `/admin/lyrics/*` | lyrics.Handler | AuthMW, AdminMW | admin | ✅ | |
| POST | `/internal/v1/lyrics/callback` | lyrics.Handler.HandleLyricsCallback | HMACMW | internal | N/A | ML callback |
| GET/POST/PUT/DELETE | `/covers/internal/*` | covers.Handler | HMACMW | internal | N/A | ML callback |
| GET | `/playlists` | playlist.Handler.ListPublicPlaylists | — | — | ✅ | |
| GET | `/playlists/:id` | playlist.Handler.GetPlaylist | OptionalAuthMW | — | ✅ | |
| POST | `/playlists` | playlist.Handler.CreatePlaylist | AuthMW | user | ✅ | |
| GET | `/playlists/me` | playlist.Handler.ListMyPlaylists | AuthMW | user | ✅ | |
| PUT/DELETE | `/playlists/:id` | playlist.Handler | AuthMW | user | ✅ | |
| PUT | `/playlists/:id/collaborative` | playlist.Handler.SetCollaborative | AuthMW | user | ✅ | |
| GET/POST/DELETE/PUT | `/playlists/:id/tracks/*` | playlist.Handler | AuthMW | user | ✅ | |
| GET/POST/DELETE | `/library/tracks/like` | library.Handler.LikeTrack/UnlikeTrack | AuthMW | user | ✅ | POST/DELETE (should be PUT/DELETE) |
| GET/POST/DELETE | `/library/albums/like` | library.Handler.LikeAlbum/UnlikeAlbum | AuthMW | user | ✅ | Same |
| GET/POST/DELETE | `/library/artists/follow` | library.Handler.FollowArtist/UnfollowArtist | AuthMW | user | ✅ | Duplicates `/artists/:id/follow` |
| POST/GET | `/library/history` | library.Handler.AddPlayHistory/ListPlayHistory | AuthMW | user | ✅ | Duplicates `/history` |
| GET | `/library/recently-played` | library.Handler.ListRecentlyPlayed | AuthMW | user | ✅ | Duplicates `/recommendations/recent` |
| POST/GET/DELETE | `/history/record` | history.Handler.RecordPlay/GetHistory | AuthMW | user | ✅ | |
| DELETE | `/history/:id` | history.Handler.DeleteHistoryItem | AuthMW | user | ✅ | |
| DELETE | `/history` | history.Handler.ClearHistory | AuthMW | user | ✅ | |
| POST/DELETE | `/artists/:id/follow` | follow.Handler.FollowArtist/UnfollowArtist | AuthMW | user | ✅ | Duplicates `/library/artists/follow` |
| POST/DELETE | `/users/:id/follow` | follow.Handler.FollowUser/UnfollowUser | AuthMW | user | ✅ | |
| GET/POST | `/recommendations/*` | recommendation.Handler | AuthMW | user | ✅ | |
| GET/POST | `/analytics/*` | analytics.Handler | — | — | ✅ | Anonymous! No auth on `/analytics/event` |
| GET/POST | `/admin/analytics/*` | analytics.Handler | AuthMW, AdminMW | admin | ✅ | |
| GET/POST | `/notifications/*` | notification.Handler | AuthMW | user | ✅ | |
| GET/POST | `/moderation/*` | moderation.Handler | AuthMW, ModeratorMW | moderator+ | ✅ | |
| GET/POST | `/admin/dashboard/*` | dashboard.Handler | AuthMW, AdminMW | admin | ✅ | |
| GET/POST | `/contribution/*` | contribution.Handler | AuthMW, ModeratorMW, AdminMW | moderator+ | ✅ | |
| GET/POST | `/import/*` | importcmd.Handler | AuthMW, AdminMW | admin | ✅ | |
| GET/POST | `/subscription/*` | subscription.Handler | AuthMW (except `/plans`) | user | ✅ | `/checkout` no idempotency key |
| GET | `/player/tracks/:id` | player.Handler.GetPlaybackTrack | — | — | ✅ | **No auth** — public stream metadata |
| GET | `/player/tracks/:id/stream` | player.Handler.StreamTrack | — | — | ⚠️ | **No auth, no Range support** |
| GET | `/player/tracks/:id/cover` | player.Handler.StreamCoverTrack | — | — | ✅ | No auth |
| GET | `/admin/player/tracks/:id` | player.Handler.GetAdminPlaybackTrack | AuthMW | user | ✅ | |
| GET | `/admin/player/tracks/:id/stream` | player.Handler.StreamAdminTrack | AuthMW | user | ✅ | |
| GET/POST | `/search` | search.Handler | OptionalAuth, RateLimitOptional(60) | — | ✅ | |
| GET | `/catalog/search` | search.Handler.Search | OptionalAuth, RateLimitOptional(60) | — | ✅ | Alias |
| POST | `/reactions/*` | reactions.Handler | AuthMW | user | ✅ | |
| GET/POST | `/gamification/*` | gamification.Handler | AuthMW, AdminMW | user/admin | ✅ | |
| GET/POST | `/creator/*` | creator.Handler | AuthMW | user | ✅ | |
| GET/POST | `/social/*` | social.Handler | AuthMW | user | ✅ | |
| GET/POST | `/ai/*` | ai.Handler | AuthMW | user | ✅ | |
| GET/POST | `/cacheadmin/*` | cacheadmin.Handler | AuthMW, AdminMW | admin | ✅ | |
| GET/POST | `/video/*` | video.Handler | AuthMW, AdminMW | user/admin | ✅ | |
| GET/POST | `/video/internal/*` | video.Handler | HMACMW | internal | N/A | ML callback |
| GET | `/ws` | WSHub.HandleWebSocket | Token in query param | user | N/A | WebSocket |

---

## Swagger Mismatches

| Route in Code | Route in Swagger | Issue |
|---------------|------------------|-------|
| `/lyrics/tracks/:trackId/lyrics` (GET) | `/api/v1/admin/lyrics` (POST) | Doubled prefix; wrong path/method |
| `/player/tracks/:id/stream` (GET) | `/api/v1/player/tracks/{id}/stream` | Missing `security: Bearer` |
| `/albums` (POST) | `/albums` (POST) | Missing `security: Bearer` in Swagger |
| `/artists` (POST) | `/artists` (POST) | Missing `security: Bearer` in Swagger |
| `/genres` (POST) | `/genres` (POST) | Missing `security: Bearer` in Swagger |
| `/tracks` (POST) | `/tracks` (POST) | Missing `security: Bearer` in Swagger (but `/admin/tracks` has it) |
| `/analytics/event` (POST) | `/analytics/event` | No auth in code or Swagger — anonymous endpoint |

---

## Routes in Swagger but Not in Code (or Vice Versa)

- Swagger has `/admin/media/upload` (POST) — matches code
- Swagger has `/admin/tracks` (GET) — matches code (admin)
- Swagger has `/albums` (GET, POST) — POST missing auth
- Swagger has `/tracks` (GET) — public list, OK
- **Missing from Swagger:** `/catalog/*` public routes (mounted at `/api/v1/catalog/...` but Swagger basePath is `/api/v1` so should be `/api/v1/catalog/...`)
- **Missing from Swagger:** `/library/*` all routes
- **Missing from Swagger:** `/history/*`, `/playlists/*` (except `/playlists` GET), `/follow/*`, `/subscription/*`, `/player/*` (some)

---

*Generated from `internal/app/routes.go` and module `routes.go` files. Cite file:line for each claim above.*