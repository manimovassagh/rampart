# Adapter Package Publishing Design

## Overview

Publish 7 Rampart adapter packages to their respective registries (6 initially, Maven Central deferred), improve code quality for v0.1.0, and add CI automation for future releases.

## Prerequisites

Before implementation:
- **npm**: Register `@rampart` organization on npmjs.com (check availability first)
- **PyPI**: Verify `rampart-python` name is available
- **Go**: No registration needed — auto-published via tags
- **Maven**: Deferred — requires Sonatype OSSRH registration

## Packages

| Package | Registry | Install Command | Directory |
|---------|----------|----------------|-----------|
| `@rampart/web` | npm | `npm install @rampart/web` | `adapters/frontend/web` |
| `@rampart/react` | npm | `npm install @rampart/react` | `adapters/frontend/react` |
| `@rampart/nextjs` | npm | `npm install @rampart/nextjs` | `adapters/frontend/nextjs` |
| `@rampart/node` | npm | `npm install @rampart/node` | `adapters/backend/node` |
| `github.com/manimovassagh/rampart/adapters/backend/go` | Go modules | `go get github.com/manimovassagh/rampart/adapters/backend/go` | `adapters/backend/go` |
| `rampart-python` | PyPI | `pip install rampart-python` | `adapters/backend/python` |
| `com.rampart:rampart-spring-boot-starter` | Maven Central | Maven/Gradle dependency | `adapters/backend/spring` |

## Licensing

- Server (Rampart core): AGPL-3.0 (unchanged)
- All adapter packages: MIT — allows proprietary use without AGPL obligations
- Each adapter directory gets its own MIT LICENSE file

## Phase 1: Publishing Blockers

### 1.1 All Packages — MIT LICENSE File

Add a standard MIT LICENSE file to each of the 7 adapter directories.

### 1.2 npm Packages — Publishing Config

Add to each npm package.json:
```json
"publishConfig": { "access": "public" }
```

Scoped packages (`@rampart/*`) default to restricted on npm. This makes them public.

### 1.3 npm Packages — File Filtering

All 4 npm packages already have `"files": ["dist"]` in package.json, which acts as an allowlist. npm automatically includes `package.json`, `README.md`, and `LICENSE` alongside the `files` entries. No `.npmignore` is needed.

### 1.4 Fix Inter-Package Dependencies

`@rampart/react` has:
```json
"@rampart/web": "file:../web"
```

Change to:
```json
"@rampart/web": "^0.1.0"
```

`@rampart/nextjs` has peer dependencies on `@rampart/react` and `@rampart/web` (correct — these are optional companion packages, not hard runtime dependencies).

### 1.5 Publish Order (npm)

Due to inter-package dependencies, npm packages must be published in this order:
1. `@rampart/web` (no Rampart dependencies)
2. `@rampart/node` (no Rampart dependencies)
3. `@rampart/react` (depends on `@rampart/web`)
4. `@rampart/nextjs` (peer deps on `@rampart/web` + `@rampart/react`)

### 1.6 Missing READMEs

Create README.md for:
- `@rampart/web` — document `RampartClient`, `loginWithRedirect()`, `handleCallback()`, `authFetch()`, `getUser()`
- `@rampart/react` — document `RampartProvider`, `useAuth()`, `ProtectedRoute`

Existing READMEs (node, nextjs, python) are adequate for v0.1.0.

### 1.7 Go Module

Go modules publish automatically when tagged. Convention for subdirectory modules:
```
git tag adapters/backend/go/v0.1.0
git push origin adapters/backend/go/v0.1.0
```

### 1.8 Python Package

Has pyproject.toml with hatchling backend. Needs LICENSE file (1.1). Ready for:
```
cd adapters/backend/python
python -m build
twine upload dist/*
```

### 1.9 Maven Central (Deferred)

Requires Sonatype OSSRH account registration, GPG key setup, and `~/.m2/settings.xml` configuration. The pom.xml already has instructions. Deferred until user has Sonatype credentials.

## Phase 2: Code Quality for v0.1.0

### 2.1 @rampart/web — Token Expiry Check

`isAuthenticated()` (line 214 of client.ts) currently only checks `this.tokens?.access_token != null` without verifying expiry. Fix to also check the `exp` claim:
```typescript
isAuthenticated(): boolean {
  if (!this.tokens) return false;
  const payload = this.decodeToken(this.tokens.access_token);
  if (!payload?.exp) return false;
  return payload.exp * 1000 > Date.now();
}
```

### 2.2 @rampart/react — Fix ProtectedRoute

All three return paths in `protected-route.ts` (lines 21, 25, 36) use `createElement("div", ...)`. Change all to use React Fragment to avoid unnecessary DOM nesting.

### 2.3 @rampart/nextjs — Add Tests

Add basic test coverage for:
- `withRampartAuth()` middleware — public path matching, redirect on missing token
- `validateToken()` — null on invalid token
- `useRampartSession()` — fetch user on mount

### 2.4 Error Type Consistency

The existing `RampartError` in `@rampart/web` uses OAuth 2.0 error format (`error`, `error_description`). Keep this convention across all packages for consistency with RFC 6749. Do not rename fields.

## Phase 3: CI Workflow

### 3.1 Workflow: `.github/workflows/publish-adapters.yml`

Triggered on tag push. Tag patterns determine which package to publish.

Tag format: `<package-name>@<version>` (no `v` prefix for npm/PyPI, `v` prefix for Go convention).

```yaml
on:
  push:
    tags:
      - '@rampart/web@*'
      - '@rampart/react@*'
      - '@rampart/nextjs@*'
      - '@rampart/node@*'
      - 'rampart-python@*'
      - 'adapters/backend/go/v*'
```

Tag-to-directory mapping:

| Tag Pattern | Directory | Action |
|------------|-----------|--------|
| `@rampart/web@*` | `adapters/frontend/web` | npm publish |
| `@rampart/react@*` | `adapters/frontend/react` | npm publish |
| `@rampart/nextjs@*` | `adapters/frontend/nextjs` | npm publish |
| `@rampart/node@*` | `adapters/backend/node` | npm publish |
| `rampart-python@*` | `adapters/backend/python` | twine upload |
| `adapters/backend/go/v*` | — | No action (Go proxy auto-fetches) |

Jobs:
1. **npm-publish**: Extract package name from tag, map to directory, `npm ci && npm run build && npm publish --provenance`
2. **pypi-publish**: `python -m build && twine upload dist/*`
3. **go-module**: No CI job needed — Go proxy picks up tags automatically

Use `--provenance` flag for npm publish to add supply chain attestation.

### 3.2 Required GitHub Secrets

| Secret | Registry | How to Get |
|--------|----------|-----------|
| `NPM_TOKEN` | npm | `npm token create` or npmjs.com settings |
| `PYPI_TOKEN` | PyPI | pypi.org account → API tokens |

Maven Central secrets (`OSSRH_USERNAME`, `OSSRH_PASSWORD`, `GPG_PRIVATE_KEY`) deferred.

## Rollback Plan

If a broken version is published:
- **npm**: `npm unpublish @rampart/<pkg>@<version>` (within 72 hours), or publish a patch fix
- **PyPI**: Yank the release via pypi.org UI, publish a patch fix
- **Go**: Add `retract` directive to `go.mod` and tag a new version

## Implementation Order

1. Verify name availability: `@rampart` on npm, `rampart-python` on PyPI
2. Add MIT LICENSE to all 7 adapter directories
3. Add publishConfig to 4 npm package.json files
4. Fix @rampart/react file: dependency
5. Write READMEs for @rampart/web and @rampart/react
6. Code quality fixes (expiry check, ProtectedRoute, nextjs tests)
7. Add CI publish workflow
8. Commit, create PR, merge
9. Register `@rampart` npm org (manual)
10. Publish in order: @rampart/web, @rampart/node, @rampart/react, @rampart/nextjs
11. Tag and publish Go module
12. Build and publish to PyPI
13. Maven Central: deferred until Sonatype credentials available

## Success Criteria

- All 6 packages installable from their registries (Maven deferred)
- `npm install @rampart/web @rampart/react @rampart/nextjs @rampart/node` works
- `go get github.com/manimovassagh/rampart/adapters/backend/go` works
- `pip install rampart-python` works
- CI workflow publishes on tag push
- All packages have LICENSE, README, and pass their tests
