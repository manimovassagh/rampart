# Contributing to Rampart

Thanks for your interest in contributing to Rampart! This document covers the basics you need to get started.

## Prerequisites

- **Go 1.25+**
- **PostgreSQL 16+**
- **Redis 7+**
- **Docker & Docker Compose** (for running the full stack locally)

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

# Start dependencies (Postgres + Redis)
docker compose -f docker-compose.dev.yml up -d

# Run tests
make test

# Run all quality checks (lint, vet, test, security)
make check

# Build the binary
make build
```

## Code Style

- Follow idiomatic Go. Run `gofmt` and `go vet` before committing.
- `golangci-lint` is the authoritative linter -- CI enforces it.
- Use table-driven tests. Test names should describe the scenario (e.g., `TestLogin_InvalidPassword_ReturnsUnauthorized`).
- Wrap errors with context: `fmt.Errorf("creating user: %w", err)`.
- No `interface{}`/`any` unless absolutely necessary.

## Branching & PRs

1. **Every change needs an issue first.** Open one or pick an existing one.
2. **Create a feature branch** from `main`:
   ```
   git checkout -b feat/42-short-description
   ```
3. **Keep commits small and logical.** Use conventional prefixes: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`.
4. **Open a PR** against `main`. Reference the issue: `Closes #42`.
5. **All PRs require review** from a maintainer before merging.
6. CI must pass: lint, test, security scan, build.

## What to Work On

Look for issues labeled `good first issue` or `help wanted`. If you want to tackle something larger, comment on the issue first so we can align on the approach.

## Testing

- Unit tests for all business logic.
- Integration tests for database and auth flows.
- Run the full suite: `make test` or `go test -race ./...`.
- Coverage must not drop below the established threshold.

## Security

Rampart is an IAM product -- security is the product. Please read `SECURITY.md` for our vulnerability reporting policy. When contributing:

- Never commit secrets, passwords, or tokens.
- Use well-known crypto libraries -- never roll your own.
- Validate all input at system boundaries.
- Test both happy paths and attack vectors for auth flows.

## Questions?

Open a discussion or comment on the relevant issue. We're happy to help.
