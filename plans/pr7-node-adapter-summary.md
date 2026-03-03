# PR #7 — Node.js & Web Adapters: Implementation Summary

> **Branch:** `feat/node-adapter`
> **Status:** Implementation complete, integration-tested against running Rampart server
> **Base:** `main` (after PR #35 OIDC Foundation merged)

## What Was Built

### 1. `@rampart/node` — Backend Adapter (`adapters/backend/node/`)

Express middleware that verifies Rampart JWTs and sets typed `req.auth`.

**Source files:**
- `src/types.ts` — `RampartConfig`, `RampartClaims`, `RampartError`, Express `req.auth` augmentation
- `src/middleware.ts` — `rampartAuth()` factory, `mapClaims()`, `sendUnauthorized()`
- `src/index.ts` — barrel exports

**Key design:**
- Uses `jose.createRemoteJWKSet()` for lazy JWKS fetching + caching + key rotation
- Derives JWKS URL deterministically: `issuer + "/.well-known/jwks.json"` (no discovery fetch)
- Error messages match Go server exactly (`auth.go:37,43,49`)
- Error JSON matches `apierror.Error` format: `{error, error_description, status}`
- `given_name`/`family_name` only set when present (matches Go `omitempty`)

**Tests:** 9 passing (vitest) — local JWKS server with `jose` key generation
**Build:** dual ESM/CJS via tsup → `dist/index.js` + `dist/index.cjs` + `dist/index.d.ts`

### 2. `@rampart/web` — Frontend Adapter (`adapters/frontend/web/`)

Framework-agnostic browser client for Rampart auth.

**Source files:**
- `src/types.ts` — `RampartClientConfig`, `RampartTokens`, `LoginRequest`, `RegisterRequest`, `LoginResponse`, `RampartUser`, `RampartError`
- `src/client.ts` — `RampartClient` class
- `src/index.ts` — barrel exports

**RampartClient API:**
- `login(req)` → `LoginResponse` (stores tokens)
- `register(req)` → `RampartUser`
- `refresh()` → `RampartTokens` (uses stored refresh token)
- `logout()` → void (invalidates on server, clears locally)
- `getUser()` → `RampartUser` (calls `/me`)
- `authFetch(url, init?)` → `Response` (auto-attaches Bearer, retries on 401 after refresh)
- `getAccessToken()`, `getTokens()`, `isAuthenticated()`, `setTokens()`
- `onTokenChange` callback for external persistence (e.g. localStorage)

**Tests:** 9 passing (vitest) — mock HTTP server
**Build:** dual ESM/CJS via tsup

### 3. Sample Apps (`samples/`)

**`samples/express-backend/`** — Express API (port 3001)
- `GET /api/health` — public
- `GET /api/profile` — protected, returns user info from `req.auth`
- `GET /api/claims` — protected, returns raw JWT claims
- Uses `file:` link to local `@rampart/node`

**`samples/web-frontend/`** — Vite + vanilla TypeScript (port 3000)
- Login form → calls Rampart `/login` via `@rampart/web`
- Profile fetch → calls Express `/api/profile` with `authFetch()`
- Logout → calls Rampart `/logout`
- Token persistence via `localStorage`
- Vite proxy: `/api` → `localhost:3001`

### 4. Claude Code Skill (`.claude/skills/rampart-node-setup/`)

Invocable via `/rampart-node-setup <issuer-url>`. Guides users through:
- Installing `@rampart/node`
- Adding middleware to Express app
- Available claims on `req.auth`
- Frontend token handling pattern

## Folder Structure

```
adapters/
├── README.md                          # Index: backend vs frontend
├── backend/
│   └── node/                          # @rampart/node (Express middleware)
│       ├── src/{types,middleware,index}.ts
│       ├── __tests__/middleware.test.ts
│       ├── package.json, tsconfig, tsup, vitest
│       └── README.md
└── frontend/
    └── web/                           # @rampart/web (browser client)
        ├── src/{types,client,index}.ts
        ├── __tests__/client.test.ts
        └── package.json, tsconfig, tsup, vitest

samples/
├── README.md                          # Quick start + architecture diagram
├── express-backend/                   # Sample API using @rampart/node
│   ├── src/server.ts
│   └── package.json, tsconfig
└── web-frontend/                      # Sample UI using @rampart/web
    ├── src/main.ts
    ├── index.html
    └── package.json, tsconfig, vite.config
```

## Integration Test Results (Against Running Rampart)

All tests passed against Rampart on `localhost:8080`:

| Test | Result |
|------|--------|
| `GET /api/health` (public) | 200 `{"status":"ok"}` |
| `GET /api/profile` (no token) | 401 `"Missing authorization header."` |
| `GET /api/profile` (invalid token) | 401 `"Invalid or expired access token."` |
| `GET /api/profile` (Basic auth) | 401 `"Invalid authorization header format."` |
| Login to Rampart → use token on `/api/profile` | 200 with all claims |
| Login to Rampart → use token on `/api/claims` | 200 with raw JWT payload |

## Commits on Branch

1. `feat: add @rampart/node and @rampart/web adapter packages` — both adapters, 18 tests, skill file
2. `feat: add sample apps and reorganize adapter folder structure` — samples, backend/frontend categorization

## What's Next

- [ ] Create PR to merge `feat/node-adapter` → `main`
- [ ] Consider `@rampart/react` hooks adapter (wraps `@rampart/web` with `useAuth()` context)
- [ ] Consider `@rampart/angular` and `@rampart/vue` adapters
- [ ] Publish to npm when ready
- [ ] Add `audience` validation option when Rampart supports `aud` claim
- [ ] Add README for `@rampart/web`

## Reference Files (Go Server)

| File | What's Mirrored |
|------|-----------------|
| `internal/middleware/auth.go:37,43,49` | Three error messages |
| `internal/middleware/auth.go:92-111` | Error JSON format |
| `internal/token/token.go:16-24` | Claims struct → RampartClaims |
| `internal/handler/discovery.go:26` | JWKS URI derivation |
| `internal/apierror/error.go:16-21` | `{error, error_description, status}` shape |
| `internal/handler/login.go` | Login response shape (access_token, refresh_token, user) |
| `internal/handler/me.go` | /me response shape |
