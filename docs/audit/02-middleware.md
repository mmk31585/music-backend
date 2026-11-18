# Audit 02 — Middleware Chain

**Source:** `internal/common/middleware/*.go`, `internal/modules/auth/middleware.go`, `internal/app/container.go`, `internal/app/http.go`

---

## Global Middleware (Applied to All Routes)

Applied in `internal/app/http.go` via `r.Use(...)` before route registration:

| Order | Middleware | Source | Purpose |
|-------|------------|--------|---------|
| 1 | `RequestID` | `middleware.RequestID` (HTTP wrapper) | Generates/propagates `X-Request-ID` header; uses UUID v4 |
| 2 | `GinZapLogger` | `middleware.GinZapLogger(log, "/health", "/metrics", "/favicon.ico")` | Structured Zap logging; skips health/metrics/favicon; adds request_id, user_id if present |
| 3 | `GinZapRecovery` | `middleware.GinZapRecovery(log, "/health", "/metrics", "/favicon.ico")` | Panic recovery with stack trace; returns 500 JSON; skips logging for health/metrics/favicon |
| 4 | `SecurityHeaders` | `middleware.SecurityHeaders()` | Sets: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `X-XSS-Protection: 1; mode=block`, `Strict-Transport-Security: max-age=31536000; includeSubDomains`, `Referrer-Policy: strict-origin-when-cross-origin`, `Permissions-Policy: camera=(), microphone=(), geolocation=()`, `Cross-Origin-Resource-Policy: same-origin`, `Cross-Origin-Opener-Policy: same-origin`, CSP with `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self' ws: wss:; media-src 'self'; frame-ancestors 'none'` |
| 5 | `CORS` | `middleware.CORS(config.CORSAllowedOrigins, log)` | Exact-origin matching (no wildcard); credentials allowed; preflight handling with `Access-Control-Max-Age: 600`; allowed headers: `Authorization,Content-Type,Accept,X-Request-ID,X-Vercel-Oidc-Token` |

**Note:** No global body size limit middleware found. Gin default is 32MB (`gin.DefaultMaxMemory` not overridden).

---

## Per-Route / Group Middleware

### Authentication Middleware (`internal/modules/auth/middleware.go`)

| Middleware | Function | Behavior | Source |
|------------|----------|----------|--------|
| `AuthMW` | `AuthMiddleware(tokenManager, fetchHierarchy)` | **Required** Bearer token; validates HS256 JWT; extracts `userId`, `email`, `role`; stores `auth_user_id`, `auth_user_role`, `auth_user_hierarchy_level` in context; calls `fetchHierarchy(roleSlug) → int` for role level | `auth/middleware.go:26-64` |
| `OptionalAuthMW` | `OptionalAuthMiddleware(tokenManager, fetchHierarchy)` | **Optional** Bearer token; if present and valid, sets same context values; if missing/invalid, continues silently | `auth/middleware.go:183-217` |
| `AdminMW` | `RequireRoleHierarchical(fetchHierarchy, "admin")` | Requires role hierarchy level ≥ admin level | `auth/middleware.go:84-139` |
| `ModeratorMW` | `RequireRoleHierarchical(fetchHierarchy, "moderator", "admin")` | Requires level ≥ moderator | `auth/middleware.go:84-139` |
| `Permissions.RequireAnyPermission(permSvc, perms...)` | Custom permission check | Delegates to `permissions.ServiceInterface` | `permissions/routes.go` |

**JWT TokenManager** (`auth/jwt.go:14-114`):
- **Algorithm:** HS256 only (`jwt.WithValidMethods([]string{"HS256"})`)
- **Secrets:** Separate `accessSecret` and `refreshSecret` (bytes)
- **TTL:** Configurable `accessTTL` (default 15min), `refreshTTL` (default 30d)
- **Claims:** `userId`, `email`, `role`, standard `jti`, `sub`, `iat`, `exp`
- **Rotation:** New JTI on each token generation; refresh token has own JTI
- **Reuse Detection:** **NOT IMPLEMENTED** — no check for refresh token reuse; old refresh tokens valid until expiry
- **Logout Invalidation:** `auth_sessions` table stores `refresh_token_hash` with `revoked_at`; `Logout` service marks revoked; but no check on access token (stateless)

### Rate Limiting (`internal/common/middleware/ratelimit.go`)

| Function | Key Strategy | Window | Limit | Behavior |
|----------|--------------|--------|-------|----------|
| `RateLimitIP(rdb, requests, window)` | `ratelimit:ip:{ip}:{method}:{path}` | configurable | `requests` | Returns 429 with `Retry-After` header |
| `RateLimitOptional(rdb, requestsPerMinute)` | User ID if authenticated (`auth_user_id`), else IP | 1 minute | `requestsPerMinute` | Falls back to IP for anonymous |

**Applied:**
- `catalog` public routes: `RateLimitOptional(rdb, 60)` (60 req/min)
- `search` routes: `RateLimitOptional(rdb, 60)`
- No global rate limit; no per-user limit on authenticated routes except catalog/search

---

## Request ID & Logging

| Component | Source | Details |
|-----------|--------|---------|
| `RequestID` | `middleware/request_id.go:14-26` | Reads `X-Request-ID` header; generates UUID v4 if missing; sets in context (`RequestIDKey`) and response header |
| `GinZapLogger` | `middleware/logger.go:15-73` | Logs: request_id, status, method, path, IP, UA, referer, latency, body_size, user_id (if auth), errors; levels: 5xx=Error, 4xx=Warn, else Info |
| `GinZapRecovery` | `middleware/logger.go:76-121` | Recovers panics; logs stack trace; returns 500 JSON; skips health/metrics/favicon |

---

## HMAC Verification (Internal/ML Callbacks)

| Middleware | Source | Purpose |
|------------|--------|---------|
| `HMACVerify` | `middleware/hmac_verify.go` | Verifies HMAC signature for internal service-to-service calls (ML service callbacks); mounted on `/internal/v1/*` routes |

---

## OIDC / Sentry / Cache

| Middleware | Source | Notes |
|------------|--------|-------|
| `OIDC` | `middleware/oidc.go` | Placeholder for Vercel OIDC token verification (not fully implemented) |
| `Sentry` | `middleware/sentry.go` | Sentry error reporting integration |
| `Cache` / `Cacher` | `middleware/cacher.go`, `cache.go` | Response caching with Redis; not used on write routes |

---

## Middleware Order Summary (Typical Authenticated Route)

```
Request → RequestID → CORS → SecurityHeaders → GinZapLogger → GinZapRecovery
       → [Group: AuthMW → Optional: AdminMW/ModeratorMW/RateLimitOptional]
       → Handler
```

---

## Gaps / Observations

| Gap | Impact |
|-----|--------|
| **No body size limit** | Gin default 32MB; large uploads could OOM |
| **No global rate limit** | Only catalog/search have optional rate limits; authenticated endpoints unbounded |
| **No refresh token reuse detection** | Rotation creates new tokens but old refresh tokens valid until expiry; no `refresh_token_reuse` detection |
| **No access token revocation** | Stateless JWT; logout only revokes refresh token in DB; access token valid until expiry (15min) |
| **Rate limit key collision** | `RateLimitOptional` uses `user:method:path` — different methods on same path share counter; could be stricter |
| **No request timeout middleware** | No per-request deadline; relies on context cancellation from client disconnect |

---

*Cite file:line for each claim. Sources: `internal/common/middleware/*.go`, `internal/modules/auth/middleware.go`, `internal/app/container.go:116-121` (middleware instantiation), `internal/app/routes.go` (route registration with middleware).*