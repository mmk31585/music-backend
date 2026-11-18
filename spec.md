# Muse Backend Security & Reliability Remediation — Execution Spec

**Version:** 1.0  
**Created:** 2026-11-18  
**Status:** Ready for Phase 0  
**Scope:** Go/Gin backend (`backend/`), PostgreSQL 17, Redis 8, OpenSearch 3.2  
**Inputs:** `docs/audit/00-summary.md` … `07-duplicates-and-inconsistencies.md`  
**Depends on:** Audit 08 (reliability), 09 (frontend map), 10 (frontend UX/perf) for Phase 7+

---

## 0. Overview

This spec defines a **phased, gated remediation workflow**. Each phase is a separate OpenCode session. Each phase has:

- A branch
- Concrete tasks
- Tests that fail before / pass after
- An exit gate
- A report: `docs/audit/phase-<N>-report.md`
- A **stop-and-review checkpoint** — no phase starts until the previous phase report is reviewed and approved

**Phase order:** 0 → 1 → 2 → 3 → 4 → 5 → 6 → (7 reliability) → (8+ frontend, UX)

---

## 1. Common Rules (paste at top of EVERY phase prompt)

```text
COMMON RULES
- Work on branch `phase-<N>-<name>`. Small commits (<~200 lines), Conventional Commits.
- Never edit shipped migrations. Add new ones starting at 000065, each with a working Down.
  Make them safe on existing data: use `CREATE INDEX CONCURRENTLY` with `-- +goose NO TRANSACTION`,
  add constraints as NOT VALID then VALIDATE.
- Before every commit: gofmt, `go vet ./...`, `golangci-lint run`,
  `go test ./... -race -count=1`.
- Every fix needs a test that FAILS before the fix and PASSES after. Show both runs.
- If the code contradicts docs/audit/*, trust the code and note it in the report.
- Do not change behavior outside this phase's scope. If a decision is ambiguous, STOP and
  list options with a recommendation instead of guessing.
- Finish with docs/audit/phase-<N>-report.md: what changed (file list), tests added,
  commands run with results, open issues. Print only that report's summary.
```

---

## 2. Audit Verdicts — What Change from Original Audit

### 2.1 Worse than rated

| Finding | Original rating | Actual severity | Evidence |
|---|---|---|---|
| `/admin/player/*` has only `AuthMW`, no `AdminMW` | Low | **Critical** — any logged-in user streams private/unreleased tracks | `player/routes.go:16-23,25-32` |
| Auth endpoints have no rate limit | Not flagged | **High** — brute force on login/register/forgot-password | `auth/routes.go`, no rate limiter |
| Deleting artist cascades into users' data | Not flagged | **High** — `ON DELETE CASCADE` wipes likes, playlists, history, queue | migrations, FK constraints |
| `Cache-Control: immutable` on predictable URL | Not flagged | **Medium** — re-uploads serve stale for up to 1h | `player/handler.go:120,158,204` |
| `listening_history` partitioned, no future partitions verified | Not flagged | **Medium** — inserts fail at month boundary if no partition | migration `000010`, `000043` |

### 2.2 Better than Swagger suggested

- Catalog writes live under `/admin/catalog/*` with `AdminMW`. "Unprotected POST /albums" is likely **stale Swagger**. Phase 0 verifies.

### 2.3 Contradictions to resolve (Phase 0)

| Claim A | Claim B |
|---|---|
| Likes and follows have no `ON CONFLICT`, return 500 (`00`) | They use `ON CONFLICT` and return 409 (`04`) |
| Refresh reuse detection is correct (`00`) | Reuse detection not implemented (`02`, `04`) |
| RequestID isn't wired (`00`) | RequestID is middleware #1 (`02`) |
| Queue/playlist code "likely single INSERT" (`04`) | Queue/playlist `max+1` races (`00`) |

### 2.4 Downgraded

- "No access-token blocklist" → **downgraded**. 15-min TTL is acceptable. Real gap: disabled user or changed role not re-checked by `AuthMW`.

---

## 3. Decisions (Baked Into Prompts)

Tell the user if any are wrong; otherwise assumed:

1. **Public tracks** stream anonymously, rate-limited, with Range support. **Private tracks** need a short-lived signed URL.
2. **`listening_history`** is canonical play table; **`followed_artists`** is canonical artist-follow table. Old endpoints delegate and get `Deprecation` headers. Nothing removed until frontend migrates.
3. **One error envelope:** `{"error":{"code","message","details","request_id"}}`. Success shapes don't change until frontend audit done.
4. **No multi-device stream limit** for now.
5. **Playlists** keep one entry per track, with caps (playlist 5000 tracks, queue 1000).
6. **Deletes are soft** (`deleted_at`).

---

## 4. Phase Overview & Effort

| Phase | Goal | Effort | Gate |
|---|---|---|---|
| 0 | Verify audit claims, safety net, CI | 1 day | `phase0-verified.md` complete; route policy test green with SECURITY-TODOs; integration helper works; CI file added |
| 1 | Security hardening | 2–3 days | All items have failing-then-passing tests; route policy test clean |
| 2 | Player/streaming | 3 days | Player package tests; manual `curl -H "Range: bytes=100-199"` returns 206 |
| 3 | Data integrity & concurrency | 4–5 days | Every race scenario has passing concurrency test; migrations apply + roll back on copy of data |
| 4 | API contract (non-breaking) | 3 days | Swagger drift test green; all list endpoints enforce limits; no frontend call site broken |
| 5 | Test suites, Schemathesis, k6 | 4–5 days | Coverage table before/after; bugs found by new tests listed |
| 6 | Performance | 3 days | Before/after table for p50/p95/p99, throughput, DB queries/request; revert non-improving changes |
| 7+ | Reliability, then frontend/UX/design | after audits 08–10 | — |

---

## 5. Phase 0 — Verify and Build the Safety Net

**Branch:** `phase-0-verify`  
**Goal:** Verify audit claims, add safety nets. **NO production behavior changes.**

### Tasks

1. **Create `docs/audit/phase0-verified.md`** — table `claim | TRUE/FALSE | evidence (file:line or failing test)` for each:
   - **a.** LikeTrack/FollowArtist (library + follow repos): `ON CONFLICT`? HTTP status on duplicate and non-existent track (FK 23503)?
   - **b.** auth Refresh: revoke+create in ONE transaction? Reuse of already-revoked token detected? Test with 20 parallel refreshes of one token: how many succeed?
   - **c.** Is `RequestID` applied to the Gin router (`internal/app/http.go` AND `cmd/api/main.go`)? Does ID appear in logs and responses?
   - **d.** playlist AddTrack/ReorderTrack and queue Add/Reorder: exact SQL, transaction, locking. Tests with 50 parallel adds.
   - **e.** `/admin/player/*` and `/admin/player/*/stream`: exact middleware chain; does a normal-user token get 200 on a private track?
   - **f.** Does AuthMiddleware check `users.is_active` or DB role, or trust JWT alone?
   - **g.** RateLimit key: raw path or `c.FullPath()`? Can varying `:id` bypass?
   - **h.** Do POST/PUT/DELETE on `/albums`, `/artists`, `/genres`, `/tracks` (non-admin) exist as real routes? Probe anonymously.
   - **i.** `/ws`: token in query string, is query logged, Origin checked?
   - **j.** subscription Checkout: real provider (Zarinpal/IDPay) or stub? Callback/verify route?
   - **k.** Who calls `player.TrackPlayed` / `IncrementPlayCount`? Reachable from any route?
   - **l.** Row counts and code refs of `play_history` vs `listening_history`, `artist_follows` vs `followed_artists`.
   - **m.** `listening_history` partitions: which exist, job/migration for future ones, INSERT with uncovered `played_at` — prove with test.
   - **n.** Columns/indexes on `queue_items` (user_id index? unique position?).

2. **Route policy test:** `internal/app/route_policy_test.go` + `testdata/route_policy.yaml`
   - Build real router with fake/stub services.
   - Enumerate every registered route.
   - Probe: anonymous request to every route not declared `public|optional|hmac` must return 401/403 (never 2xx/500).
   - Normal-user token to every `admin|moderator` route must return 403.
   - Substitute path params with random UUIDs.
   - Mark known-bad behavior in YAML with `# SECURITY-TODO(phase1)`; test allows those only.
   - Test fails when a route exists that is not in YAML.

3. **Integration test infra:** `internal/testutil` with testcontainers-go helpers (Postgres 17 + goose migrations, Redis), build tag `integration`; `make test-integration`. One sample test.

4. **CI:** `.github/workflows/ci.yml` running vet, golangci-lint, `go test -race`, integration tests, govulncheck, gitleaks (full history). Report findings but **do not fix here**.

### Exit

- `phase0-verified.md` complete
- Route policy test green with SECURITY-TODOs listed
- Integration helper works
- CI file added

---

## 6. Phase 1 — Security Hardening

**Branch:** `phase-1-security`  
**Inputs:** `phase0-verified.md`, `02-middleware.md`, `01-routes.md`

### Tasks

1. `/admin/player/*`: require `AuthMW + AdminMW` (or proper permission). Delete dead `RegisterAdminRoutes`. Test: normal user gets 403 on private track metadata and stream.
2. Rate limiting: per-IP AND per-identifier (email) on login, register, forgot-password, refresh (e.g. 10/min IP, 5/min per email); generous global per-IP on `/api/v1`; per-user on authenticated writes. Key by `c.FullPath()`, not raw path. 429 + `Retry-After`. Fail-open if Redis down (log + metric), never 500. Tests with miniredis.
3. Ingress limits: global `MaxBytesReader` (default 1 MB JSON), larger only on upload routes (existing config), `gin.MaxMultipartMemory = 8 MB`. `http.Server`: `ReadHeaderTimeout 10s`, `IdleTimeout 120s`, **NO global WriteTimeout**; use `http.ResponseController` in stream/upload handlers. Request-timeout middleware (30s) excluding stream, upload, `/ws`. Tests.
4. CORS: remove `Access-Control-Allow-Origin: *` from player handlers; rely on CORS middleware allowlist. Test allowed/disallowed origins.
5. WebSocket: stop accepting access token in query string. `POST /ws/ticket` (AuthMW) → single-use Redis ticket (30s TTL); `/ws` consumes atomically (GETDEL). Enforce Origin allowlist. Logger must not log `/ws` query strings. Tests reuse, expiry, bad origin.
6. AuthMW: reject `is_active=false` and use current DB role/hierarchy (Redis cache 60s). Test: disabled user and demoted admin rejected within TTL.
7. Refresh token rotation: atomic with reuse detection. ONE transaction: `UPDATE auth_sessions SET revoked_at=now() WHERE refresh_token_hash=$1 AND revoked_at IS NULL AND expires_at>now() RETURNING id,user_id`; zero rows ⇒ if hash exists and is revoked, treat as reuse, revoke ALL sessions of that user (`family_id` via new migration if needed); insert new session in same tx. Concurrency test: 20 parallel refreshes ⇒ exactly 1 success.
8. RequestID: ensure globally applied and in every error body and log line (fix only if Phase 0 proved broken).
9. Analytics POST: `event_type` allowlist, metadata max 2 KB / depth 3, body limit 8 KB, per-IP rate limit.
10. Config validation: refuse production start if JWT secrets < 32 bytes, equal to each other, or equal to example defaults. Verify `.env.example` has no real secrets.
11. Remove every `SECURITY-TODO(phase1)` from `route_policy.yaml`; policy test passes without exceptions.

### Exit

All items have failing-then-passing tests; route policy test clean; report lists any gitleaks/govulncheck findings.

---

## 7. Phase 2 — Player and Streaming

**Branch:** `phase-2-player`  
**Inputs:** `05-player-stream.md`

### Tasks

1. Storage interface (`internal/platform/storage`): add
   - `Stat(ctx,key) (size, etag, modTime, contentType, err)`
   - `OpenRange(ctx,key,start,end) (io.ReadCloser,error)`
   - `PresignGet(ctx,key,ttl) (string,error)`
   
   Implement for local and S3 with unit tests.

2. `StreamTrack`/`StreamAdminTrack`/cover, HEAD support:
   - Local: `http.ServeContent` (206/416/If-Range for free)
   - S3: single-range parser (`bytes=a-b`, `a-`, `-n`; multi-range ⇒ full 200 or 416) + `OpenRange`. Headers: `Accept-Ranges`, `Content-Range`, `Content-Length`, `ETag`, `Last-Modified`, 206/416/304 (`If-None-Match`).
   - `STREAM_MODE=proxy|redirect`: redirect ⇒ 302 to presigned/CDN URL.
   - `Cache-Control: public, max-age=3600` WITHOUT `immutable` unless URL has content version/etag.
   - Stop copy on client disconnect (request context); pooled buffer; don't log broken-pipe as error.
   - Soft-deleted/missing track ⇒ 404/410; track without audio ⇒ 404.

3. Access control: public ⇒ anonymous OK (rate-limited). Non-public ⇒ valid AuthMW user with rights OR signed token `?exp=&sig=` = HMAC-SHA256(secret, `trackID|userID|exp`), TTL 10 min, constant-time compare. `GetPlaybackTrack` returns correct `streamUrl` (signed for non-public). Add `canStream(ctx,user,track) error` interface with default allow-all for future premium gating.

4. Play tracking, server-authoritative:
   - `POST /api/v1/player/plays` (AuthMW) `{track_id}` ⇒ `{play_id}`; store in Redis 24h: user, track, started_at.
   - `POST /api/v1/player/plays/:playId/progress` `{position_ms, completed}` (idempotent). Count ONCE when wall-clock elapsed ≥ `min(30s, 50% duration)` and position plausible (≤ elapsed + tolerance).
   - Migration: add `play_id uuid` to `listening_history` with `UNIQUE(user_id, play_id)`. Insert `ON CONFLICT DO NOTHING`; increment `play_count` only when insert happened.
   - Keep `/history/record` and `/library/history` working, route through same service (generate `play_id`), mark `Deprecated`.

5. Tests: Range table (`0-0`, `0-`, `-500`, beyond EOF⇒416, invalid, multi-range, If-Range, HEAD), private anon⇒401/403, signed URL tamper/expired/wrong-track, client disconnect mid-stream with goleak, 50 parallel progress calls same `play_id` ⇒ exactly 1 row and `play_count+1`, short play not counted, fabricated position rejected.

### Exit

Player package has tests; manual `curl -H "Range: bytes=100-199"` returns 206 with correct headers (include output in report).

---

## 8. Phase 3 — Data Integrity and Concurrency

**Branch:** `phase-3-integrity`  
**Inputs:** `03-schema.md`, `04-concurrency.md`, `07-*.md`, `phase0-verified.md`  
**All concurrency tests run against real Postgres via `integration` tag.**

### Tasks

1. Central DB error mapping (`internal/common`): 23505⇒409, 23503⇒404/400, 40001/40P01⇒retry (helper `RetryTx(ctx, maxAttempts=3, jitter)`). Replace ad-hoc handling in all repos touched below.
2. Likes/follows: `INSERT ... ON CONFLICT DO NOTHING`; duplicate ⇒ 200/204 (idempotent), unknown track/artist ⇒ 404. Test: 100 parallel likes ⇒ exactly 1 row, zero 5xx.
3. Consolidate duplicates: canonical = `listening_history` and `followed_artists`. Migration copies missing rows from `play_history` / `artist_follows` in batches (dedupe). `library/history/follow` modules delegate to canonical repos. Add `Deprecation` + `Sunset` headers and Prometheus counter per deprecated route. Do NOT drop tables.
4. Playlist and queue ordering, one tx each:
   - Lock parent: `SELECT ... FROM playlists WHERE id=$1 FOR UPDATE`; queue: `pg_advisory_xact_lock(hashtextextended(user_id::text,0))`.
   - Migration: `uq_playlist_position DEFERRABLE INITIALLY DEFERRED`; add `UNIQUE(user_id, position) DEFERRABLE` on `queue_items` + index `(user_id, position)`.
   - Reorder = move semantics (shift range) in one tx; validate `1<=new_position<=count`; no-op unchanged; compact on delete. Caps: playlist 5000, queue 1000.
   - Check collaborative-playlist permissions on every mutation.
   - Tests: 50 parallel adds ⇒ positions exactly 1..50; concurrent reorders ⇒ no deadlock/gaps; two users editing one collaborative playlist.
5. Subscription/payments (if Phase 0 says checkout is a stub, implement DB safety only and report what provider adapter needs):
   - Partial unique index `subscriptions(user_id) WHERE status IN ('active','trialing')`.
   - `Idempotency-Key` header on `POST /subscription/checkout`; new table `idempotency_keys(user_id,key,request_hash,response,created_at) UNIQUE(user_id,key)`; same key + different body ⇒ 422.
   - Payment state machine `pending→paid|failed|canceled`, forward-only; `UNIQUE(provider, provider_payment_id)`; callback/verify replay-safe and verifies server-side with provider.
   - FK `payments.subscription_id` (NOT VALID then VALIDATE).
   - Tests: 10 parallel checkouts one key ⇒ 1 payment; different keys ⇒ only 1 active subscription; replayed callback ⇒ no double activation.
6. Background jobs: periodic enrichment only in worker (`RUN_PERIODIC_JOBS`), guard with `pg_try_advisory_lock`. Test: two instances ⇒ one runs.
7. Soft delete: migration adds `deleted_at` to tracks, albums, artists. Admin deletes set `deleted_at`; every read path filters via shared query helpers; stream ⇒ 410; user libraries/playlists keep rows (hide deleted in responses). Hard delete only via cleanup worker after 30 days. **If too big, split into 3b and report.**
8. Upload/track create: reject `audio_url`/`cover_url`/media IDs not belonging to our storage or media table; cleanup worker removes orphaned files/drafts > 24h.
9. Partitions: guarantee `listening_history` partitions for next 3 months + DEFAULT; scheduled job creates them; test insert at month boundary.
10. Constraints/indexes (new migrations): CHECK `duration_seconds>=0`, `track_number>0`, `album_type IN (...)`; indexes: `listening_history(user_id, played_at DESC)`, `liked_tracks(user_id, created_at DESC)`, `playlists(user_id, updated_at DESC)`, `notifications(user_id, created_at DESC) WHERE read_at IS NULL`. Include EXPLAIN for 5 hottest queries.

### Exit

Every race scenario has passing concurrency test; migrations apply on copy of existing data and roll back cleanly (show it).

---

## 9. Phase 4 — API Contract (Non-Breaking)

**Branch:** `phase-4-contract`  
**If frontend directory exists, grep its API calls first and list every affected call site in report.**

### Tasks

1. Errors: single envelope `{"error":{"code","message","details","request_id"}}` for ALL error responses. Keep top-level `message` key too (additive). Replace ad-hoc `c.JSON` with `internal/common/response` helpers; add lint/grep test forbidding `c.JSON(` with error status outside response package.
2. Pagination: one helper in `internal/common/pagination`: default 20, max 100 (clamp), offset max 10000; negative/non-numeric ⇒ 400. Apply to every list endpoint including `/playlists`, `/playlists/me`, library liked lists, notifications, queue. Add `meta{limit,offset,has_more}` additively where response is object.
3. Validation: email format, username `^[A-Za-z0-9_.]{3,50}$`, password 8..72 bytes, email stored lowercase + unique index `lower(email)` (new migration if missing), UUID path params ⇒ 400 malformed / 404 unknown consistently, Content-Type check on JSON routes.
4. Swagger: fix every `@Router` prefix (no doubled `/api/v1`), HTTP methods match code (document PATCH vs PUT; don't change code), add `security: Bearer` to every protected route, library params as UUID strings not integer, document all user-facing routes (target: all except internal/HMAC). CI: `swag init` + fail on `git diff docs/`; test comparing gin routes vs swagger paths, fail on undocumented route not in explicit allowlist file.
5. Duplicates: keep working but mark Deprecated in Swagger + `Deprecation` headers (done in Phase 3). Produce `docs/api-migration.md`: old ⇒ new route for frontend.
6. `/health/ready`: check Postgres, Redis, OpenSearch with 1s timeouts each; return per-dependency status.
7. Generate TypeScript client/types from `swagger.json` (openapi-typescript) into `docs/generated/`; report build command; don't wire into frontend yet.

### Exit

Swagger drift test green; all list endpoints enforce limits (tests); no frontend call site broken (list checked).

---

## 10. Phase 5 — Test Suites, Fuzzing, Load

**Branch:** `phase-5-tests`  
**Inputs:** `06-tests.md`

### Tasks

1. HTTP-level tests (httptest, real router + real middleware, Postgres/Redis via testcontainers, `integration` tag). For EACH critical route: table test covering anonymous, wrong role, IDOR (user A touching user B's playlist/queue/likes/notifications ⇒ 404), malformed input, boundary values (empty, max length, Persian + emoji), not found, duplicate, pagination bounds, response matches swagger schema.
   Priority: player, auth (register/login/refresh/logout), subscription, library, queue, playlist, history, follow, notifications, catalog album/genre/stats, upload, media, lyrics.
2. Target: critical packages ≥ 70% line coverage; every package with goroutines gets `goleak.VerifyTestMain`.
3. Go native fuzz tests: Range parser, Persian normalizer, LRC parser, search query builder (no panics, no OpenSearch DSL injection).
4. Dependency-failure tests: Redis down ⇒ rate limit fails open, cache degrades without 500; OpenSearch down ⇒ search falls back or clean 503; storage down ⇒ stream 502/503 without leaking details; DB timeout ⇒ 503 with request_id.
5. Schemathesis: `scripts/schemathesis.sh` running
   `schemathesis run docs/swagger.json --base-url http://localhost:8080/api/v1 --checks all --header "Authorization: Bearer $TOKEN"` against docker compose; build fails on any 5xx; triage in report.
6. k6 (`scripts/load/`): scenarios browse, search, stream with Range seeks, play start/progress, queue add, login; thresholds p95 < 300ms reads, error rate < 1%, 0 deadlocks in PG logs. Report numbers on this machine.
7. `make test-all` (unit, integration, fuzz smoke 30s, schemathesis); wire fast parts into CI.

### Exit

Coverage table before/after; list of bugs found by new tests (fix each in own commit or list as open issue).

---

## 11. Phase 6 — Performance (Measure First)

**Branch:** `phase-6-perf`

### Tasks

1. Add pprof (admin-only or localhost-only) and Prometheus histograms per route and per DB query group. Run k6 scenarios from Phase 5, capture CPU/heap/goroutine profiles, write `docs/audit/perf-baseline.md` with top 10 hotspots. Only fix what profiles justify.
2. play_count: Redis `INCR plays:{trackID}` on counted plays + flush worker (10–30s, batched `UPDATE ... FROM (VALUES ...)`); handle Redis loss gracefully (bounded loss acceptable, document). Popular/trending reads use flushed value.
3. Cache layer: singleflight on miss, explicit TTLs per key family, invalidation on catalog writes, negative caching for 404 (short TTL), normalized keys (query + page), bounded key cardinality, stale-while-revalidate for popular lists. Test stampede: 100 concurrent misses ⇒ 1 DB call.
4. Database: pool limits (`MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`) from config; statement/context timeouts per query class; `EXPLAIN (ANALYZE, BUFFERS)` on top 10 queries; fix N+1 (batch genres/artists per track list); keyset pagination for history/notifications.
5. Search: circuit breaker (platform/circuitbreaker has no tests: add them) + Postgres fallback when OpenSearch down.
6. HTTP: gzip for JSON only (never audio), ETag/If-None-Match on cacheable catalog GETs.
7. Streaming: confirm `STREAM_MODE=redirect` works with CDN/presigned URL; measure API CPU/bandwidth proxy vs redirect under k6 stream scenario.

### Exit

Before/after table for p50/p95/p99, throughput, DB queries/request on each optimized endpoint. Revert any change not improving a measured number.

---

## 12. Phase 7+ — Deferred

- **Phase 7, Reliability:** Needs `08-reliability.md`. Covers `safego.Go` wrapper with `recover` for every goroutine, non-blocking WebSocket hub, shutdown ordering.
- **Frontend phases:** Need `09-frontend-map.md` and `10-frontend-ux-perf.md`. Cover token-refresh single-flight, player state machine, API client from generated types, design tokens, motion rules, RTL fixes.
- **Do not start visual redesign** until Phases 1–3 land — a new look on a broken stream/token flow hides bugs.

---

## 13. Rules of Engagement

- **One fresh OpenCode session per phase.**
- Review the diff yourself after each phase.
- Paste the phase report before starting the next phase.
- **Start with Phase 0.** Send `phase0-verified.md` — it settles contradictory findings before any fixes.
