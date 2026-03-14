# Contributing to Rampart

Thanks for your interest in contributing to Rampart. This document covers everything you need to get started.

## Prerequisites

- **Go 1.26+**
- **PostgreSQL 16+**
- **Docker & Docker Compose** (for running the full stack locally)
- **golangci-lint** (installed via `make dev-setup`)

## Development Setup

```bash
# Clone the repo
git clone https://github.com/manimovassagh/rampart.git
cd rampart

# Install dev tools (golangci-lint, govulncheck, gosec)
make dev-setup

# Copy and configure environment
cp .env.example .env
# Edit .env with your local database credentials

# Start dependencies (Postgres)
docker compose up -d postgres

# Run the server
go run ./cmd/rampart
```

Alternatively, run the full stack (server + database) with Docker Compose:

```bash
docker compose up -d --build
```

## Building

```bash
# Build the server binary
go build ./cmd/rampart

# Or use the Makefile (also rebuilds Tailwind CSS)
make build
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run via Makefile
make test

# Run full quality checks (lint + vet + test + security)
make check
```

- Write unit tests for all business logic.
- Write integration tests for database and auth flows.
- Use table-driven tests where appropriate.
- Test names should describe the scenario (e.g., `TestLogin_InvalidPassword_ReturnsUnauthorized`).
- Coverage must not drop below the established threshold.

## Code Style

- Run `gofmt` and `go vet` before committing.
- `golangci-lint run` is the authoritative linter -- CI enforces it.
- Wrap errors with context: `fmt.Errorf("creating user: %w", err)`.
- Use interface-based store patterns for database access (see `internal/store/`).
- No `interface{}`/`any` unless absolutely necessary.
- Follow idiomatic Go conventions throughout.

## Branching & Pull Requests

1. **Every change needs an issue first.** Open one or pick an existing one.
2. **Fork the repo** and create a feature branch from `main`:
   ```
   git checkout -b feat/42-short-description
   ```
3. **Keep commits small and logical.** Use conventional prefixes: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`.
4. **Run tests and lint before pushing:**
   ```bash
   go test ./...
   golangci-lint run
   ```
5. **Open a PR** against `main`. Reference the issue: `Closes #42`.
6. **All PRs require review** from a maintainer before merging.
7. CI must pass: lint, test, security scan, build.

## What to Work On

Look for issues labeled `good first issue` or `help wanted`. If you want to tackle something larger, comment on the issue first so we can align on the approach.

## Security

Rampart is an IAM product -- security is the product. Please read `SECURITY.md` for our vulnerability reporting policy. When contributing:

- Never commit secrets, passwords, or tokens.
- Use well-known crypto libraries -- never roll your own.
- Validate all input at system boundaries.
- Test both happy paths and attack vectors for auth flows.

## Questions?

Open a discussion or comment on the relevant issue. We are happy to help.
