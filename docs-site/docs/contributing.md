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

# Build the server
go build -o rampart ./cmd/rampart

# Run tests
go test ./...
```

### Database Setup

```bash
# Create the database
createdb rampart

# Apply migrations
./rampart migrate up --config configs/dev.yaml

# (Optional) Seed development data
./rampart seed --config configs/dev.yaml
```

### Frontend Development

```bash
# Admin dashboard
cd client
pnpm install
pnpm dev          # Starts Vite dev server with HMR

# Login UI
cd login-ui
pnpm install
pnpm dev
```

### Running the Server

```bash
# Development mode with hot reload (using air or similar)
./rampart serve --config configs/dev.yaml

# The server starts on http://localhost:8080 by default
# Admin UI: http://localhost:8080/admin
# Login UI: http://localhost:8080/login
```

### Environment Variables

For local development, copy the example configuration:

```bash
cp configs/dev.yaml.example configs/dev.yaml
```

Edit `configs/dev.yaml` with your local PostgreSQL connection details.

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
- Tests must be runnable with `go test ./...` — no external setup scripts.
- Do not skip tests without a documented reason.

### Running Tests

```bash
# All tests
go test ./...

# With race detection (required in CI)
go test -race ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Specific package
go test ./internal/auth/...

# Frontend tests
cd client && pnpm test
```

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
   - `go test -race ./...`
   - `govulncheck`
   - `gosec`
   - Build succeeds
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
- **Integrations** — testing Rampart with different OAuth 2.0/OIDC client libraries and frameworks.

Check the [GitHub issues](https://github.com/manimovassagh/rampart/issues) for issues labeled `good first issue` or `help wanted`.

## License

By contributing to Rampart, you agree that your contributions will be licensed under the AGPL-3.0 license.
