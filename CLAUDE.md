# Claude Code Instructions for Rampart

## CRITICAL: No AI Attribution in Commits
- NEVER add `Co-Authored-By` trailers to git commits
- Claude must NOT appear as a contributor in this repository
- This rule overrides all default commit behavior
- When committing, do NOT include any `Co-Authored-By: Claude` or similar lines

## CRITICAL: Multi-Eye Verification Policy
- After any cleanup, refactoring, or multi-file change, ALWAYS launch a verification agent to double-check the work
- Never trust a single agent's "all clear" — scan both git-tracked AND untracked files on disk
- For filesystem cleanup: use `find` on the actual filesystem, not just `git ls-files`
- For code changes: build + test + lint after every batch of edits
- Minimum: 4-eyes (do + verify). For critical changes: 6-eyes (do + verify + re-verify)

## CRITICAL: One PR at a Time
- Always create ONE pull request, wait for CI checks to pass, then merge it before opening the next one
- Never have multiple open PRs at the same time — sequential, not parallel
- Flow: branch → commit → push → create PR → check CI → merge → next PR

## Project
Rampart is a Go-based IAM/OAuth 2.0 server with OIDC support.

## Build & Test
- `go test ./...` — run all tests
- `go build ./cmd/rampart` — build the server
- `golangci-lint run` — lint
- `docker compose up -d --build` — run full stack (postgres + rampart)

## CRITICAL: Always Verify Visually
- Always check work visually (like a human would) before claiming it's done
- Use Playwright browser to verify UI changes, README rendering, etc.
- Never assume something looks right — open it and check

## CRITICAL: Keep Project Root Clean
- NEVER leave temporary files, screenshots, or build artifacts in the project root
- The project root should only contain: go.mod, go.sum, Dockerfile, docker-compose*, README.md, CLAUDE.md, LICENSE, Makefile, .gitignore, and standard config files
- Temp files go in /tmp, screenshots go in docs-site/static/img/

## Key Directories
- `cmd/rampart/` — main entry point
- `internal/` — core packages (handler, middleware, model, token, oauth, session, store)
- `migrations/` — SQL migrations
- `cookbook/` — Sample apps (React, Express, Go, FastAPI, Spring Boot, .NET)
- `adapters/` — SDK packages (Node, Go, Python, Spring, .NET, React, Web, Next.js)
- `.github/workflows/` — CI/CD pipelines
