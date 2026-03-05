# Plan: PR #2 — User Registration + Registration Page

## Context

PR #1 (server foundation) is merged. Rampart has a working HTTP server, PostgreSQL connection, migrations (organizations + users tables), middleware chain, health endpoints, and CI pipeline. **This PR adds the first real feature**: user self-registration with a React frontend.

**Branch:** `feat/6-user-registration`
**Closes:** #6

## Scope

### Backend
- User model (Go struct matching users table)
- Argon2id password hashing (hash + verify)
- Input validation (email, password policy, username format)
- Database queries (CreateUser, GetUserByEmail, GetUserByUsername, GetDefaultOrganizationID)
- Registration handler (`POST /register`)
- New apierror helpers (Conflict 409, ValidationError 400 with fields)

### Frontend
- React + Vite + Tailwind registration page in `web/`
- Form with: username, email, password, first name, last name
- Client-side validation + server error display
- Vite dev proxy to Go backend on :8080

### What's deferred (not this PR)
- Email verification (needs email service)
- Admin CRUD endpoints (`/api/v1/admin/users`)
- Frontend embedding in Go binary
- Rate limiting on registration

## New Dependency

`golang.org/x/crypto` — provides `argon2` package for argon2id hashing. BSD-licensed, Go sub-repo.

## Files to Create

```
internal/
  model/
    user.go                    # User struct, RegistrationRequest, UserResponse, ToResponse()
  auth/
    password.go                # HashPassword(), VerifyPassword() — argon2id
    password_test.go
    validate.go                # ValidateEmail(), ValidatePassword(), ValidateUsername(), ValidateRegistration()
    validate_test.go
  handler/
    register.go                # RegisterHandler, POST /register
    register_test.go
  database/
    user.go                    # CreateUser, GetUserByEmail, GetUserByUsername on *DB
    organization.go            # GetDefaultOrganizationID, GetOrganizationIDBySlug on *DB
web/
    package.json, vite.config.ts, tsconfig.json, index.html
    src/
      main.tsx, App.tsx
      components/RegistrationForm.tsx
      api/register.ts
      types/index.ts
```

## Files to Modify

```
internal/apierror/error.go      # Add Conflict(), WriteValidation(), FieldError type
internal/apierror/error_test.go  # Tests for new helpers
internal/server/routes.go        # Add RegisterAuthRoutes()
cmd/rampart/main.go              # Wire RegisterHandler
go.mod / go.sum                  # Add golang.org/x/crypto
.gitignore                       # Add web/node_modules, web/dist
```

## Key Design Decisions

**Password hashing:** Argon2id with OWASP params (3 iterations, 64MB memory, 4 threads, 32-byte key, 16-byte salt). PHC string format for storage.

**Model package:** Separate `internal/model/` avoids coupling domain types to handler or database. `PasswordHash` tagged `json:"-"` — can never leak. Separate `UserResponse` struct enforces this at the type level.

**Auth package:** `internal/auth/` for password hashing + validation. Leaf package with no internal deps. Will grow to hold token/session logic later.

**DB queries on `*database.DB`:** Follow the existing pattern (like `Ping()`). Handler defines a `UserStore` interface (like `Pinger`), DB struct satisfies it.

**Default org:** Self-registered users go to the "default" organization (seeded by migration 000001). Handler looks up org UUID via `GetDefaultOrganizationID()`.

**Timing safety:** Every registration response takes at least 250ms (deferred sleep) to prevent user enumeration via timing analysis.

**Request body limit:** `http.MaxBytesReader` at 1MB prevents memory exhaustion.

**Frontend separate in dev:** Vite dev server on :3000 proxies API calls to Go on :8080. Embedding in Go binary deferred to PR #5.

## Implementation Order

1. Create branch `feat/6-user-registration`
2. `go get golang.org/x/crypto` + `go mod tidy`
3. Create `internal/model/user.go` — User, RegistrationRequest, UserResponse structs
4. Create `internal/auth/password.go` + tests — argon2id hash/verify
5. Create `internal/auth/validate.go` + tests — email/password/username validation
6. Add Conflict() and WriteValidation() to `internal/apierror/error.go` + tests
7. Create `internal/database/user.go` — CreateUser, GetUserByEmail, GetUserByUsername
8. Create `internal/database/organization.go` — GetDefaultOrganizationID
9. Create `internal/handler/register.go` + tests — POST /register handler
10. Add RegisterAuthRoutes() to `internal/server/routes.go`
11. Wire in `cmd/rampart/main.go`
12. Verify backend: `go build`, `go test`, `golangci-lint`, curl tests
13. Set up `web/` — React + Vite + Tailwind
14. Build RegistrationForm component
15. Verify frontend: `npm run dev`, test form submission
16. Update `.gitignore`
17. Push, create PR

## Package Dependency Graph (Updated)

```
cmd/rampart/main.go
  ├── internal/config        (no deps)
  ├── internal/model         (depends on: google/uuid)
  ├── internal/auth          (depends on: x/crypto — leaf package)
  ├── internal/database      (depends on: model)
  ├── internal/server        (depends on: middleware)
  ├── internal/handler       (depends on: apierror, auth, model — DB via UserStore interface)
  ├── internal/middleware     (no internal deps)
  └── internal/apierror      (depends on: middleware for HeaderRequestID)
```

No circular dependencies.

## Verification

- `go build ./...` compiles
- `go test -race ./internal/...` passes all tests
- `golangci-lint run ./...` clean
- `curl -X POST localhost:8080/register` with valid data → 201 + user JSON
- `curl` with weak password → 400 + field errors
- `curl` with duplicate email → 409 conflict
- Response never contains `password_hash`
- Response takes ~250ms regardless of outcome (timing safety)
- `cd web && npm run dev` → registration form at localhost:3000
- Form submission proxies to backend, shows success or errors
- CI passes all 4 jobs
