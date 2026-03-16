---
sidebar_position: 5
title: AI-Powered Setup
description: Use Claude Code skills to set up Rampart authentication in any app in seconds. Auto-detect your stack, install SDKs, and wire up auth end-to-end.
---

# AI-Powered Setup

Rampart ships with built-in [Claude Code](https://claude.com/claude-code) skills that can set up authentication in any application in seconds. Just type a slash command and the skill auto-detects your project, installs the right SDK, and wires up auth end-to-end.

No other IAM product offers AI-powered setup. This is a unique Rampart differentiator.

## How It Works

1. Open your project in Claude Code
2. Type the skill command (e.g., `/rampart-react-setup`)
3. The skill scans your project, detects your framework, and applies the correct configuration
4. You get a fully working auth integration with OAuth 2.0 PKCE, JWT validation, protected routes, and more

Each skill creates production-ready code — not scaffolding. You get proper error handling, typed claims, environment variable configuration, and Rampart-compatible error responses out of the box.

## Available Skills

### Stack-Specific Setup

| Command | What It Does |
|---------|-------------|
| `/rampart-react-setup` | Adds OAuth 2.0 PKCE login flow to a React SPA. Creates `AuthProvider`, `useAuth()` hook, `ProtectedRoute` component, and OAuth callback handler using `@rampart-auth/react`. |
| `/rampart-node-setup` | Adds JWT validation middleware to an Express app. Installs `@rampart-auth/node`, protects routes, and provides typed `req.auth` claims. |
| `/rampart-nextjs-setup` | Sets up server-side JWT validation, Next.js middleware auth guards, API route protection, and client-side auth context using `jose`. |
| `/rampart-go-setup` | Adds JWKS-based JWT middleware to a Go backend (net/http, chi, gin, or fiber). Creates typed claims and Rampart-compatible error responses. |
| `/rampart-python-setup` | Adds JWT verification to FastAPI or Flask. Auto-detects your framework, creates auth dependencies/decorators, and sets up JWKS validation. |
| `/rampart-spring-setup` | Configures Spring Security OAuth2 Resource Server with Rampart's JWKS. Sets up `SecurityFilterChain`, claims helper, and `@PreAuthorize` support. |

### Infrastructure & Operations

| Command | What It Does |
|---------|-------------|
| `/rampart-docker-quickstart` | Creates a `docker-compose.yml` with Rampart and PostgreSQL. Configures health checks, creates an admin user, and registers an OAuth client. |
| `/rampart-fullstack-secure` | The all-in-one skill. Auto-detects your entire stack (frontend + backend), spins up Rampart if needed, and wires up authentication end-to-end across all layers. |
| `/rampart-ci-check` | Runs the full CI pipeline locally before pushing. Executes lint, test, vet, security scan, and cross-compile checks — exactly as GitHub Actions would. |

## Example Usage

### Adding auth to a React app

```
> /rampart-react-setup http://localhost:8080
```

The skill will:
1. Install `@rampart-auth/react` (or `jose` as a fallback)
2. Create `src/auth/config.ts` with your issuer URL
3. Create `src/auth/AuthProvider.tsx` with login/logout/token state
4. Create `src/auth/Callback.tsx` to handle the OAuth redirect
5. Create `src/auth/ProtectedRoute.tsx` for guarding routes
6. Wire everything into your app's router
7. Add environment variables to `.env`

### Adding auth to an Express backend

```
> /rampart-node-setup http://localhost:8080
```

The skill will:
1. Install `@rampart-auth/node` and `jose`
2. Add `rampartAuth()` middleware to your Express app
3. Give you typed `req.auth` with user claims (sub, email, org_id, etc.)
4. Configure Rampart-standard 401 JSON error responses

### Securing an entire full-stack app

```
> /rampart-fullstack-secure
```

The skill will:
1. Scan your project for `package.json`, `go.mod`, `pom.xml`, `requirements.txt`, etc.
2. Detect your frontend and backend frameworks
3. Spin up Rampart with Docker Compose if no instance is running
4. Apply the correct backend skill (Node, Go, Python, Spring, Next.js)
5. Apply the correct frontend skill (React, Next.js, or adapted patterns)
6. Register an OAuth client and configure environment variables
7. Provide a summary with test instructions

## Optional Argument

Most skills accept an optional issuer URL argument:

```
> /rampart-react-setup https://auth.mycompany.com
```

If omitted, skills default to `http://localhost:8080` or read from environment variables (`RAMPART_ISSUER`).

## Requirements

- [Claude Code](https://claude.com/claude-code) installed and running
- The Rampart repository cloned (skills are loaded from `.claude/skills/`)
- For Docker skills: Docker and Docker Compose installed
- For stack-specific skills: the relevant package manager (npm, pip, go, maven/gradle)
