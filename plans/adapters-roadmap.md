# Rampart Adapters — Master Roadmap

> This file is the single source of truth for all adapter/SDK work.
> Future sessions: read this first before touching adapters/ or samples/.

## Vision

Rampart is not another auth library — it's **the IAM platform companies switch to from Keycloak**.
The adapters are what developers touch every day. They must be **better than Auth0, Clerk, and Keycloak SDKs** — not "as good", better. Better DX, better types, better docs, better samples, shinier UI components.

### What "better" means concretely

**vs Auth0** — simpler setup (no tenant config, no dashboard clicking), self-hosted, no vendor lock-in
**vs Clerk** — same beautiful UX but open-source, self-hosted, no per-user pricing
**vs Keycloak** — TypeScript-first SDKs (not Java adapters), modern React components (not FreeMarker), single binary

### The bar for every adapter

1. **Zero-config start** — `rampartAuth({ issuer })` and you're done. No client IDs, no secrets, no dashboard setup for basic use.
2. **Standalone + providers** — Works with just Rampart, or with Google/GitHub/Microsoft/SAML. Same API for both.
3. **RBAC built-in** — `requireRoles("admin")` middleware, `hasRole()` on the client, `<ProtectedRoute roles={["admin"]}>` in React.
4. **Pre-built UI components** — Drop-in `<LoginForm>`, `<SignUpForm>`, `<UserButton>`, `<OrgSwitcher>` (like Clerk). Themeable, accessible, beautiful by default.
5. **E2E samples** — Every adapter ships with a working sample showing the FULL lifecycle: unauth 401 → signup → login → protected API → role check → social login → logout.
6. **Production-grade** — Rate limiting awareness, token rotation, PKCE by default, XSS-safe token storage, CSRF protection.
7. **Perfect TypeScript** — Every response typed, every error typed, autocomplete everywhere, no `any`.

### Competitive feature matrix (target state)

| Feature | Auth0 | Clerk | Keycloak | **Rampart** |
|---------|-------|-------|----------|-------------|
| Self-hosted | No | No | Yes | **Yes** |
| Open-source core | No | No | Yes | **Yes** |
| TypeScript-first SDKs | Yes | Yes | No | **Yes** |
| Pre-built UI components | Limited | Excellent | Terrible | **Excellent** |
| Social login (Google, GitHub, etc.) | Yes | Yes | Yes | **Yes** (planned) |
| RBAC | Yes | Yes | Yes | **Yes** (planned) |
| MFA (TOTP, WebAuthn) | Yes | Yes | Yes | **Yes** (planned) |
| Organization/multi-tenant | Yes | Yes | Yes | **Yes** (built-in) |
| SAML/SCIM | Enterprise | Enterprise | Yes | **Enterprise** (planned) |
| Per-user pricing | $$$$ | $$$$ | Free | **Free** |
| Setup time | 30 min | 10 min | 2 hours | **5 min** (target) |
| Single binary deploy | N/A | N/A | No (Java) | **Yes** |
| Beautiful default theme | OK | Great | Awful | **Great** (target) |
| Zero-config quickstart | No | Almost | No | **Yes** |

## Current State (as of PR #40)

### Packages

| Package | Location | Status | Tests |
|---------|----------|--------|-------|
| `@rampart/node` | `adapters/backend/node/` | Shipped | 9/9 |
| `@rampart/web` | `adapters/frontend/web/` | Shipped | 9/9 |

### What Works Today

**Backend (`@rampart/node`):**
- `rampartAuth({ issuer })` Express middleware
- RS256 JWT verification via `jose.createRemoteJWKSet()`
- Typed `req.auth: RampartClaims` (sub, email, org_id, preferred_username, email_verified, given_name?, family_name?, roles?)
- 401 errors match Go server format: `{error, error_description, status}`
- Dual ESM/CJS build

**Frontend (`@rampart/web`):**
- `RampartClient` class — framework-agnostic
- `login()`, `register()`, `logout()`, `refresh()`, `getUser()`
- `authFetch()` with automatic 401 → refresh → retry
- `onTokenChange` callback for persistence
- `isAuthenticated()`, `getAccessToken()`, `setTokens()`

**Samples (`samples/`):**
- `express-backend/` — Express API (port 3001) with protected routes
- `web-frontend/` — Vite + vanilla TS (port 3000) with full UI:
  - "Try without auth" → shows 401 rejection
  - Signup form with validation errors
  - Login → profile + email verification badge
  - Multiple endpoint buttons (profile, claims, /me)
  - Logout

### Folder Structure

```
adapters/
├── README.md
├── backend/
│   └── node/                     # @rampart/node
│       ├── src/types.ts          # RampartConfig, RampartClaims, RampartError
│       ├── src/middleware.ts      # rampartAuth(), mapClaims(), sendUnauthorized()
│       ├── src/index.ts
│       └── __tests__/middleware.test.ts
└── frontend/
    └── web/                      # @rampart/web
        ├── src/types.ts          # RampartClientConfig, RampartTokens, LoginRequest, etc.
        ├── src/client.ts         # RampartClient class
        ├── src/index.ts
        └── __tests__/client.test.ts

samples/
├── README.md
├── express-backend/              # Uses @rampart/node
│   └── src/server.ts
└── web-frontend/                 # Uses @rampart/web
    ├── index.html
    └── src/main.ts
```

## Roadmap

### Phase 1: RBAC (when Go server adds roles) ────────────────────

**Go server changes needed:**
- Add `roles []string json:"roles,omitempty"` to `token.Claims`
- Add role assignment API (admin endpoint)
- Include roles in JWT on login

**Adapter changes:**
- `roles?: string[]` already on `RampartClaims` and `RampartUser` ✅
- Add `requireRoles(...roles: string[])` middleware to `@rampart/node`:
  ```typescript
  // Usage: only allow admins
  app.get("/admin", rampartAuth({ issuer }), requireRoles("admin"), handler);
  ```
- Add `hasRole(role: string): boolean` to `RampartClient`
- Sample: add admin-only route to express-backend, show 403 in web-frontend

### Phase 2: Authorization Code + PKCE (social login) ───────────

This is needed for Google, GitHub, Microsoft, and any third-party OAuth provider.

**Go server changes needed:**
- `GET /authorize` — starts auth code flow, redirects to provider or shows login
- `POST /token` — exchanges auth code for tokens (standard OAuth 2.0 token endpoint)
- `GET /callback` — handles provider callback, creates session
- Provider configuration in admin API (client_id, client_secret per provider)

**`@rampart/web` additions:**
```typescript
// Redirect to Rampart's authorize endpoint
client.loginWithRedirect({
  provider: "google",          // optional — omit for Rampart's own login page
  redirectUri: "/callback",
  scope: "openid profile email",
});

// Handle the callback (exchange code for tokens)
await client.handleCallback();  // reads code from URL, exchanges for tokens

// PKCE is automatic — generates code_verifier/code_challenge internally
```

**`@rampart/node` — no changes needed.** It just verifies JWTs regardless of grant type.

**Sample additions:**
- Add "Login with Google" button to web-frontend
- Add callback route handling
- Show that the same `@rampart/node` middleware works for both direct login and social login tokens

### Phase 3: Framework-specific adapters ─────────────────────────

All wrap `@rampart/web` — thin layers adding framework-native patterns.

**`@rampart/react`** (`adapters/frontend/react/`):
```typescript
// Provider
<RampartProvider issuer="http://localhost:8080">
  <App />
</RampartProvider>

// Hook
const { user, login, logout, isAuthenticated, isLoading } = useAuth();

// Protected route component
<ProtectedRoute roles={["admin"]}>
  <AdminPage />
</ProtectedRoute>
```

**`@rampart/vue`** (`adapters/frontend/vue/`):
```typescript
// Plugin
app.use(rampartPlugin, { issuer: "http://localhost:8080" });

// Composable
const { user, login, logout } = useAuth();

// Route guard
router.beforeEach(authGuard);
```

**`@rampart/angular`** (`adapters/frontend/angular/`):
```typescript
// Module
RampartModule.forRoot({ issuer: "http://localhost:8080" })

// Service injection
constructor(private auth: RampartService) {}

// Route guard
canActivate: [RampartAuthGuard]
```

**Samples for each:**
- `samples/react-app/` — Vite + React + `@rampart/react`
- `samples/vue-app/` — Vite + Vue + `@rampart/vue`
- `samples/angular-app/` — Angular + `@rampart/angular`

### Phase 4: Additional backend adapters ─────────────────────────

**`@rampart/go`** (Go middleware):
```go
r.Use(rampart.Auth(rampart.Config{Issuer: "http://localhost:8080"}))
// req.Context() contains rampart.Claims
```

**`@rampart/python`** (FastAPI/Flask):
```python
@app.get("/protected")
async def protected(claims: RampartClaims = Depends(rampart_auth)):
    return {"user": claims.sub}
```

## Industry Best Practices Checklist

These apply to ALL adapters — check before shipping each one:

### Security
- [ ] Tokens stored in memory by default (not localStorage) — consumer opts in to persistence
- [ ] PKCE for all browser-based auth code flows (no implicit grant)
- [ ] Token refresh happens automatically on 401 (single retry)
- [ ] Refresh token sent to server on logout (server-side revocation)
- [ ] No sensitive data in error messages
- [ ] JWKS cached with automatic rotation support

### TypeScript
- [ ] Full type definitions shipped (.d.ts)
- [ ] Strict mode compatible
- [ ] Generic types where appropriate (e.g., custom claims extension)
- [ ] JSDoc on all public APIs

### Build
- [ ] Dual ESM/CJS output
- [ ] Tree-shakeable
- [ ] Zero or minimal runtime dependencies
- [ ] Peer dependencies for frameworks (not bundled)

### Testing
- [ ] Unit tests with real crypto (not mocked JWT verification)
- [ ] Integration tests against mock server (not Rampart dependency)
- [ ] E2E sample that demonstrates full lifecycle
- [ ] Error cases tested (401, expired, wrong issuer, missing fields)

### Documentation
- [ ] README with install, quick start, API reference
- [ ] Working sample in samples/ directory
- [ ] Error message reference

## Reference: Go Server Endpoints

| Endpoint | Method | Auth | Purpose |
|----------|--------|------|---------|
| `/register` | POST | No | Create user account |
| `/login` | POST | No | Direct login → tokens |
| `/token/refresh` | POST | No | Refresh access token |
| `/logout` | POST | No | Revoke refresh token |
| `/me` | GET | Bearer | Current user profile |
| `/.well-known/openid-configuration` | GET | No | OIDC Discovery |
| `/.well-known/jwks.json` | GET | No | Public signing keys |

**Future endpoints (for social login):**
| `/authorize` | GET | No | Start auth code flow |
| `/token` | POST | No | Exchange code for tokens |
| `/callback` | GET | No | Provider callback handler |

## Reference: Error Format

All 401 responses from adapters match the Go server:
```json
{
  "error": "unauthorized",
  "error_description": "Missing authorization header.",
  "status": 401
}
```

Three error messages (from `internal/middleware/auth.go`):
1. `"Missing authorization header."` — no Authorization header
2. `"Invalid authorization header format."` — not Bearer scheme
3. `"Invalid or expired access token."` — verification failed

Future 403 for RBAC:
```json
{
  "error": "forbidden",
  "error_description": "Insufficient permissions.",
  "status": 403
}
```

## Reference: JWT Claims (from `internal/token/token.go`)

```json
{
  "iss": "http://localhost:8080",
  "sub": "user-uuid",
  "iat": 1772578000,
  "exp": 1772578900,
  "org_id": "org-uuid",
  "preferred_username": "jane",
  "email": "jane@example.com",
  "email_verified": true,
  "given_name": "Jane",           // omitempty
  "family_name": "Doe",           // omitempty
  "roles": ["admin", "editor"]    // future — omitempty
}
```
