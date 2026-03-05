# CI/CD Pre-Push Check

Run the full CI pipeline locally before pushing. This catches lint errors, test failures, security issues, and build problems before they reach GitHub Actions.

## When to use

Run this before `git push` or when you want to verify everything passes locally, exactly as CI would check it.

## Steps

1. Run all checks in parallel where possible:

```bash
# Go vet
go vet ./...

# gofmt check (should produce no output)
gofmt -l .

# golangci-lint (must be v2)
golangci-lint run --timeout=5m

# Tests with race detector
go test -race -count=1 ./...

# Security: govulncheck
govulncheck ./...

# Cross-compile check (linux/amd64)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /dev/null ./cmd/rampart
```

2. If docs-site was modified, also build Docusaurus:

```bash
cd docs-site && npm run build
```

3. Report results clearly: which checks passed, which failed, and exact error messages for failures.

## Fix common issues

- **errcheck**: Add `_, _ =` before fmt.Fprint* calls writing to tabwriter/stderr
- **gocritic/octalLiteral**: Use `0o700` not `0700`
- **gocritic/paramTypeCombine**: Use `func(a, b any)` not `func(a any, b any)`
- **revive/error-strings**: Error messages must not start with capital letter or end with punctuation
- **gosec/G304**: Add `//nolint:gosec` with justification for file paths from trusted sources
- **gosec/G704**: Add `//nolint:gosec` with justification for URLs from config, not user input
