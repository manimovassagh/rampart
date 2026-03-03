# Plan: PR #3 — Login + JWT Sessions

## Context

PR #2 added user registration, the registration page, and a default admin user (`admin`/`admin`). But there's no way to actually **log in** yet — the "Sign in" link on the registration page goes nowhere. This PR completes the basic auth loop: login with credentials, get JWT tokens, and see a login page in the frontend.

**Branch:** `feat/login-sessions`
**Closes:** #9 (partially — login page, consent screen deferred)

## Scope

### Backend
- JWT access token generation + verification (HS256, `golang-jwt/jwt/v5`)
- Refresh token generation + storage in PostgreSQL `sessions` table
- `POST /login` handler — accepts email/username + password, returns tokens
- `POST /token/refresh` handler — exchange refresh token for new access token
- `POST /logout` handler — invalidate refresh token
- Auth middleware — extracts + verifies Bearer token, stores user in context
- `GET /me` — protected endpoint returning current user (proves auth works)
- New apierror helpers: `Unauthorized()`, `Forbidden()`
- Update `last_login_at` on successful login

### Frontend
- Login page component (same professional style as registration)
- Simple client-side routing (hash-based: `#/login`, `#/register`, `#/dashboard`)
- After login success: store tokens, show minimal dashboard with "Welcome, {username}"
- Token-aware API client (attaches Bearer header)
- Logout button

### What's deferred
- Redis session store (use PostgreSQL for now, swap later)
- Token rotation on refresh
- OAuth 2.0 flows (separate PR #7)

## New Dependency

`github.com/golang-jwt/jwt/v5` — most widely used Go JWT library. MIT-licensed.

## Files to Create

```
internal/
  token/
    token.go              # GenerateAccessToken, GenerateRefreshToken, VerifyAccessToken, Claims
    token_test.go
  session/
    store.go              # SessionStore interface + PostgreSQL implementation
  handler/
    login.go              # LoginHandler: POST /login, POST /token/refresh, POST /logout
    login_test.go
    me.go                 # MeHandler: GET /me (protected)
    me_test.go
  middleware/
    auth.go               # Auth middleware — Bearer token verification
    auth_test.go
migrations/
  000004_create_sessions.up.sql
  000004_create_sessions.down.sql
web/src/
  components/LoginForm.tsx
  components/Dashboard.tsx
  api/auth.ts             # login, refresh, logout, getMe API calls
```

## Files to Modify

```
internal/config/config.go       # Add JWTSecret, AccessTokenTTL, RefreshTokenTTL
internal/config/config_test.go  # Tests for new config fields
internal/apierror/error.go      # Add Unauthorized(), Forbidden()
internal/apierror/error_test.go # Tests
internal/server/routes.go       # Add login/refresh/logout/me routes
cmd/rampart/main.go             # Wire LoginHandler, MeHandler, session store, auth middleware
go.mod / go.sum                 # Add golang-jwt/jwt/v5
web/src/App.tsx                 # Hash router with login/register/dashboard views
web/src/types/index.ts          # Add LoginRequest, LoginResponse, TokenPair types
web/vite.config.ts              # Add proxy for new endpoints
```

## Key Design Decisions

**JWT algorithm:** HS256 (HMAC-SHA256) with a shared secret. Appropriate for a monolithic server. Will migrate to RS256 (asymmetric) when OIDC discovery (PR #8) requires publishing a JWKS endpoint.

**Token types:**
- **Access token** (15 min): Stateless JWT with user claims. Short-lived so compromise window is small.
- **Refresh token** (7 days): Opaque JWT stored hashed in PostgreSQL `sessions` table. Revocable via logout/admin action.

**Claims (OIDC-aligned):**
```
sub: user UUID
org_id: organization UUID
preferred_username: username
email: email
email_verified: bool
given_name: first name
family_name: last name
iat: issued at
exp: expiration
```

**Session storage: PostgreSQL** (not Redis yet). The `sessions` table stores SHA-256 hashes of refresh tokens. This avoids adding a Redis dependency for now — swap to Redis later when performance matters.

**Auth middleware:** Extracts `Authorization: Bearer <token>`, verifies signature + expiration, stores `AuthenticatedUser` in request context. Protected routes use `r.Group()` with this middleware.

**Login identifier:** Accept either `email` or `username` (like Keycloak). Handler tries email first, falls back to username. Generic "Invalid credentials" error for both missing user and wrong password.

**Timing safety:** Same 250ms floor as registration handler.

**Config:** 3 new env vars with sensible defaults:
- `RAMPART_JWT_SECRET` (required, min 32 bytes)
- `RAMPART_ACCESS_TOKEN_TTL` (default: 900 = 15 min)
- `RAMPART_REFRESH_TOKEN_TTL` (default: 604800 = 7 days)

## Implementation Order

1. Create branch `feat/login-sessions`
2. `go get github.com/golang-jwt/jwt/v5` + `go mod tidy`
3. Add JWT config fields to `internal/config/config.go`
4. Create `migrations/000004_create_sessions.{up,down}.sql`
5. Create `internal/token/token.go` + tests — JWT generation & verification
6. Create `internal/session/store.go` — SessionStore interface + PG implementation
7. Add `Unauthorized()` and `Forbidden()` to `internal/apierror/`
8. Create `internal/handler/login.go` + tests — POST /login, POST /token/refresh, POST /logout
9. Create `internal/middleware/auth.go` + tests — Bearer token middleware
10. Create `internal/handler/me.go` + tests — GET /me (protected)
11. Update `internal/server/routes.go` — register all new routes
12. Wire everything in `cmd/rampart/main.go`
13. Verify backend: `go build`, `go test -race`, `golangci-lint`
14. Add frontend types + API client for auth
15. Build LoginForm component (matching registration page style)
16. Build minimal Dashboard component with logout
17. Add hash-based routing in App.tsx
18. Verify with Playwright MCP: login flow, dashboard, logout
