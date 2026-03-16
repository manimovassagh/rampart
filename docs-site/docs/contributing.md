---
sidebar_position: 99
title: Contributing
description: How to contribute to Rampart — development setup, code style, testing, pull request process, and security reporting.
---

# Contributing to Rampart

Rampart is an open-source project and contributions are welcome. This guide covers everything you need to get started: development environment setup, coding standards, testing requirements, and the pull request process.

## Code of Conduct

Be respectful, constructive, and professional. We are building security infrastructure — quality, precision, and thoughtful communication matter.

## Development Setup

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22+ | Backend development |
| PostgreSQL | 15+ | Primary data store |
| Node.js | 20+ | Admin UI and login UI development |
| pnpm | 9+ | Frontend package manager |
| Git | 2.40+ | Version control |

### Clone and Build

```bash
# Clone the repository
git clone https://github.com/manimovassagh/rampart.git
cd rampart

# Install Go dependencies
go mod download

# Rebuild admin console CSS (required before first build)
make admin-css

# Build the server (rebuilds CSS automatically, outputs to bin/rampart)
make build

# Run all quality gates (formatting, vet, lint, tests, security)
make check
```

### Database Setup

```bash
# Create the database
createdb rampart

# Set the database URL
export RAMPART_DB_URL="postgres://localhost:5432/rampart?sslmode=disable"

# Or use Docker Compose (recommended — handles database automatically)
docker compose up -d
```

### Running the Server

```bash
# Build and run the server
make run

# Or run directly after building (configured via environment variables)
export RAMPART_DB_URL="postgres://localhost:5432/rampart?sslmode=disable"
./bin/rampart

# The server starts on http://localhost:8080 by default
# Admin UI: http://localhost:8080/admin
# Login UI: http://localhost:8080/login
```

### Environment Variables

Rampart is configured entirely through environment variables (see [Configuration](../getting-started/configuration.md)). For local development, set at minimum:

```bash
export RAMPART_DB_URL="postgres://localhost:5432/rampart?sslmode=disable"
export RAMPART_ISSUER="http://localhost:8080"
```

## Code Style

### Go

Rampart follows standard Go conventions enforced by tooling:

- **`gofmt`** — all Go code must be formatted with `gofmt`. No exceptions.
- **`go vet`** — must pass with no warnings.
- **`golangci-lint`** — the project uses golangci-lint with a strict configuration. Run it before committing:

```bash
golangci-lint run ./...
```

### Style Guidelines

- **Function names** — use `camelCase`, no underscores. Test functions follow `TestFunctionName_Scenario_ExpectedResult` format but use camelCase within each segment.
- **Error handling** — always wrap errors with context: `fmt.Errorf("creating user: %w", err)`. Never swallow errors silently.
- **No `interface{}` or `any`** unless absolutely necessary. Use strong types.
- **No dead code** — do not commit commented-out code or unused functions.
- **String constants** — extract string literals used 3+ times into named constants.
- **Keep functions short** — if a function needs a comment explaining what it does, consider splitting it.

### Frontend (htmx + Go Templates)

- The admin UI and login UI use htmx with Go server-side templates, styled with Tailwind CSS.
- Follow the existing template conventions in the `internal/` handler and template directories.
- Keep JavaScript minimal — prefer htmx attributes for interactivity.
- When modifying admin CSS, use `make admin-css` to rebuild or `make admin-css-watch` for live development.

## Testing Requirements

### What Must Be Tested

- **All business logic** in `internal/` packages must have unit tests.
- **All auth flows** must have integration tests covering both success and failure paths.
- **Database operations** must have integration tests using a real PostgreSQL instance (not mocks).
- **API endpoints** must have HTTP-level tests validating request/response contracts.

### Test Conventions

- Use **table-driven tests** (idiomatic Go):

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "user@example.com", false},
        {"missing domain", "user@", true},
        {"empty string", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
            }
        })
    }
}
```

- **Test names** describe the scenario: `TestLoginFlow_InvalidPassword_ReturnsUnauthorized`.
- Use the standard `testing` package. Avoid testify or other assertion libraries unless already in use.
- Tests must be runnable with `go test -race -count=1 ./...` — no external setup scripts.
- Do not skip tests without a documented reason.

### Running Tests

```bash
# All tests (with race detection and cache bypass — matches CI)
go test -race -count=1 ./...

# Or use Make (equivalent to the above)
make test

# With coverage report
make test-cover

# Enforce coverage threshold
make test-threshold

# Specific package
go test -race -count=1 ./internal/handler/...

# Run all quality gates (the canonical CI command)
make check
```

## Cookbook (Sample Apps)

The `cookbook/` directory contains complete sample applications demonstrating how to integrate with Rampart:

| Directory | Description |
|-----------|-------------|
| `cookbook/react-app` | React frontend using OIDC/OAuth 2.0 |
| `cookbook/express-backend` | Express.js backend with token validation |
| `cookbook/fastapi-backend` | FastAPI (Python) backend integration |
| `cookbook/go-backend` | Go backend integration |
| `cookbook/spring-backend` | Spring Boot (Java) backend integration |
| `cookbook/dotnet-backend` | .NET backend integration |
| `cookbook/web-frontend` | Vanilla web frontend example |

These are useful references when developing new features or testing changes to Rampart's auth flows.

## Adapter Development

The `adapters/` directory contains official client libraries (SDKs) for integrating applications with Rampart. Adapters are published as packages for their respective ecosystems.

### Backend Adapters

| Directory | Language/Framework | Package |
|-----------|--------------------|---------|
| `adapters/backend/node` | Node.js | npm |
| `adapters/backend/python` | Python | PyPI |
| `adapters/backend/go` | Go | Go module |
| `adapters/backend/dotnet` | .NET | NuGet |
| `adapters/backend/spring` | Spring Boot | Maven |

### Frontend Adapters

| Directory | Framework | Package |
|-----------|-----------|---------|
| `adapters/frontend/react` | React | npm |
| `adapters/frontend/nextjs` | Next.js | npm |
| `adapters/frontend/web` | Vanilla JS | npm |

### Testing Adapters

Adapters have their own CI pipeline defined in `.github/workflows/adapters-ci.yml`, which runs automatically when files under `adapters/` are changed. When developing an adapter:

1. Follow the conventions of the existing adapters in the same language.
2. Include unit tests in the adapter package.
3. Ensure your changes pass the adapters CI pipeline before submitting a PR.

## Pull Request Process

### Before You Start

1. **Check for an existing issue.** Every change should be connected to a GitHub issue. If one does not exist, create it first.
2. **Discuss large changes.** For significant features or architectural changes, open an issue for discussion before writing code.

### Branch Naming

Create a branch from `main` with a descriptive name:

```
feat/42-oidc-auth-code-flow
fix/57-token-expiry-off-by-one
chore/update-golangci-lint
docs/api-endpoint-reference
test/refresh-token-rotation
refactor/session-store-interface
```

The format is `type/issue-number-short-description`. Include the issue number when applicable.

### Commit Messages

Use conventional commit prefixes:

```
feat: add PKCE support to authorization endpoint
fix: correct token expiry calculation for refresh tokens
test: add integration tests for client credentials flow
refactor: extract session validation into middleware
chore: update Go to 1.22.1
docs: add CORS configuration examples
```

Commit messages should be clear and concise. Write them as a human developer would — no verbose explanations or AI-generated filler.

### Submitting a PR

1. **Push your branch** and open a pull request against `main`.
2. **Fill in the PR template:**
   - What changed and why.
   - Which issue this closes (e.g., `Closes #42`).
   - How to test the change.
   - Any breaking changes or migration steps.
3. **Ensure all CI checks pass:**
   - `golangci-lint`
   - `go vet`
   - `go test -race -count=1 ./...`
   - `govulncheck`
   - `gosec`
   - Build succeeds
   - Or simply run `make check` locally before pushing.
4. **Wait for review.** All PRs must be reviewed and approved by the project owner before merging. Do not self-merge.

### PR Guidelines

- Keep PRs focused. One logical change per PR.
- Small PRs are reviewed faster than large ones.
- If a PR grows too large, split it into sequential PRs.
- Respond to review feedback promptly and respectfully.
- Rebase on `main` if your branch falls behind.
- Delete your branch after merge.

### What Will Be Reviewed

- Correctness and completeness of the implementation.
- Test coverage for new and changed code.
- Security implications (this is an IAM product — every change is security-relevant).
- Code style and consistency with the existing codebase.
- Documentation for new public APIs or configuration options.
- Performance impact for hot paths (token validation, session lookup).

## Security Reporting

**Do not report security vulnerabilities through public GitHub issues.**

If you discover a security vulnerability in Rampart:

1. Check the repository's `SECURITY.md` file for the current security contact.
2. Send a detailed report including:
   - Description of the vulnerability.
   - Steps to reproduce.
   - Potential impact assessment.
   - Suggested fix (if you have one).
3. You will receive acknowledgment within 48 hours.
4. A resolution timeline will be communicated within 7 days.
5. Security reporters will be credited in the release notes (unless they prefer anonymity).

We follow responsible disclosure. Please give us reasonable time to address the issue before public disclosure.

## Areas Where Help Is Needed

If you are looking for ways to contribute, these areas are particularly valuable:

- **Test coverage** — improving test coverage for existing packages.
- **Documentation** — improving API documentation and adding usage examples.
- **Security review** — reviewing auth flows, crypto usage, and input validation.
- **Performance** — profiling and optimizing hot paths.
- **Frontend** — improving the admin dashboard and login UI.
- **Adapters** — developing and improving client SDKs in `adapters/`.
- **Integrations** — testing Rampart with different OAuth 2.0/OIDC client libraries and frameworks.

Check the [GitHub issues](https://github.com/manimovassagh/rampart/issues) for issues labeled `good first issue` or `help wanted`.

## License

By contributing to Rampart, you agree that your contributions will be licensed under the MIT license.
