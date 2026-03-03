# Plan: Phase 1 — PR #1 Server Foundation

> Saved: 2026-03-03
> Branch: `feat/server-foundation`
> Closes: #1, #3, #4, #5, #11

## Phase 1 PR Breakdown

| PR | Branch | Scope | Closes |
|----|--------|-------|--------|
| **#1 (this plan)** | `feat/server-foundation` | CI, branch protection, HTTP server, config, DB, migrations, Docker | #1, #3, #4, #5, #11 |
| #2 | `feat/6-user-registration` | User model, argon2id hashing, registration endpoint | #6 |
| #3 | `feat/7-oauth-authz-code` | OAuth 2.0 Auth Code + PKCE, client model, sessions, tokens | #7 |
| #4 | `feat/8-oidc-discovery` | OIDC ID tokens, discovery, JWKS, userinfo | #8 |
| #5 | `feat/9-login-ui` | Login/consent React + Vite SPA | #9 |
| #6 | `feat/10-admin-dashboard` | Admin dashboard React + Vite SPA | #10 |
| #7 | `feat/2-readme-license` | LICENSE file, README updates | #2 |

## PR #1 Scope — What We're Building

### Pre-code: Branch Protection (gh CLI)
- Protect `main` branch: require PR reviews, require status checks (lint, test, build), no force push
- Add `CODEOWNERS` file: `* @manimovassagh`

### Files to Create/Modify

```
.github/
  workflows/ci.yml          # 4 parallel jobs: lint, test (with PG), security, build
  CODEOWNERS                # * @manimovassagh
cmd/
  rampart/
    main.go                 # Entry point — load config, connect DB, run migrations, start server
internal/
  config/
    config.go               # Config struct + Load() from env vars
    config_test.go
  database/
    database.go             # pgx pool: Connect(), Ping()
    database_test.go
    migrations.go           # golang-migrate runner
  server/
    server.go               # HTTP server with graceful shutdown (10s deadline)
    server_test.go
    routes.go               # chi router + middleware chain
  middleware/
    requestid.go            # X-Request-Id generation/propagation
    requestid_test.go
    logging.go              # Structured request logging (slog)
    logging_test.go
    recovery.go             # Panic recovery
    recovery_test.go
  handler/
    health.go               # GET /healthz (liveness), GET /readyz (readiness)
    health_test.go
  apierror/
    error.go                # Consistent JSON error format per docs/api/overview.md
    error_test.go
migrations/
  000001_create_organizations.up.sql
  000001_create_organizations.down.sql
  000002_create_users.up.sql
  000002_create_users.down.sql
Dockerfile                  # Multi-stage: golang builder → distroless nonroot
docker-compose.yml          # rampart + postgres:16 + redis:7
.golangci.yml               # Linter config
```

**Modified:** `Makefile` (build target → `./cmd/rampart`), `main.go` (deleted, moved to `cmd/`)

### Dependencies to Add

| Package | Purpose |
|---------|---------|
| `github.com/go-chi/chi/v5` | HTTP router (net/http compatible, lightweight) |
| `github.com/go-chi/cors` | CORS middleware |
| `github.com/jackc/pgx/v5` | PostgreSQL driver + connection pool |
| `github.com/golang-migrate/migrate/v4` | SQL migration runner |
| `github.com/google/uuid` | UUID generation for request IDs |

### Architecture — Package Dependency Graph

```
cmd/rampart/main.go
  ├── internal/config       (no deps)
  ├── internal/database     (depends on: config)
  ├── internal/server       (depends on: config, database, handler, middleware)
  ├── internal/handler      (depends on: database, apierror)
  ├── internal/middleware    (no internal deps — stdlib only)
  └── internal/apierror     (depends on: middleware for request ID)
```

No circular dependencies. `middleware` and `apierror` are leaf packages.

### Key Design Decisions

**Config:** Env vars only for now (RAMPART_DB_URL, RAMPART_PORT, etc.). YAML support later.

**Server:** chi router + stdlib `net/http.Server`. Timeouts set: Read 10s, Write 30s, Idle 60s (Slowloris protection). Graceful shutdown with 10s deadline via `http.Server.Shutdown()`.

**Middleware order:** RequestID → Recovery → CORS → Logging → RealIP. Request ID first so all logging has it. Recovery second to catch panics.

**Database:** pgx pool (MaxConns=25, MinConns=2). Migrations run at startup (like Keycloak/Zitadel). Uses plain SQL files, not Go code.

**Initial schema:** Organizations + users tables only. Matches `docs/architecture/data-model.md`. Default "default" org seeded. Multi-tenant indexes: `UNIQUE(email, org_id)`, `UNIQUE(username, org_id)`.

**Docker:** Multi-stage build → distroless nonroot image (~15MB). docker-compose with PG + Redis health checks.

**CI:** 4 parallel GitHub Actions jobs. PG service container for integration tests. Actions pinned by SHA.

### Implementation Order

1. Set up branch protection via `gh` CLI
2. Create branch `feat/server-foundation`
3. Move `main.go` → `cmd/rampart/main.go`, update Makefile, verify `make build`
4. Add `internal/config/` + tests
5. Add `internal/apierror/` + tests
6. Add `internal/middleware/` (requestid, logging, recovery) + tests
7. Add chi, create `internal/server/` (server.go, routes.go) + tests
8. Add pgx, create `internal/database/database.go` + tests
9. Add golang-migrate, create `internal/database/migrations.go` + `migrations/*.sql`
10. Add `internal/handler/health.go` + tests
11. Wire everything in `cmd/rampart/main.go`
12. Verify: `docker compose up` → hit `/healthz` and `/readyz`
13. Add `.github/workflows/ci.yml`, `CODEOWNERS`, `.golangci.yml`
14. Run `make check`, fix any issues
15. Push, create PR closing #1, #3, #4, #5, #11

## Verification

- `make build` compiles successfully
- `make test` passes all tests (unit + integration with local PG)
- `make lint` passes with `.golangci.yml` config
- `docker compose up` starts all 3 services, rampart connects to PG and Redis
- `curl localhost:8080/healthz` → `{"status":"alive"}`
- `curl localhost:8080/readyz` → `{"status":"ready"}`
- GitHub Actions CI passes all 4 jobs on the PR
- Migrations create organizations + users tables correctly
- Default organization is seeded
