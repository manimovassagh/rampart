---
name: rampart-fullstack-secure
description: Secure an entire full-stack application with Rampart IAM. Detects your tech stack (frontend + backend), spins up Rampart, and wires up authentication end-to-end. The one-stop skill for adding auth to any project.
argument-hint: [issuer-url]
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Glob, Grep, Agent
---

# Secure a Full-Stack App with Rampart

Automatically detect your tech stack and add end-to-end authentication powered by Rampart.

## What This Skill Does

1. Scans your project to detect frontend and backend frameworks
2. Spins up Rampart (or connects to an existing instance)
3. Configures backend JWT validation middleware
4. Configures frontend OAuth 2.0 PKCE login flow
5. Registers the OAuth client in Rampart
6. Wires everything together end-to-end

## Detection Rules

### Backend Detection

Scan the project for these indicators:

| Framework | Detection | Skill to use |
|-----------|-----------|-------------|
| **Express/Node.js** | `package.json` with `express`, or `require("express")` / `import express` | `/rampart-node-setup` |
| **Next.js** | `package.json` with `next`, or `next.config.*` exists | `/rampart-nextjs-setup` |
| **FastAPI** | `requirements.txt`/`pyproject.toml` with `fastapi`, or `from fastapi import` | `/rampart-python-setup` (FastAPI) |
| **Flask** | `requirements.txt`/`pyproject.toml` with `flask`, or `from flask import` | `/rampart-python-setup` (Flask) |
| **Go** | `go.mod` exists, `net/http` or chi/gin/fiber imports | `/rampart-go-setup` |
| **Spring Boot** | `pom.xml`/`build.gradle` with `spring-boot`, `@SpringBootApplication` | `/rampart-spring-setup` |

### Frontend Detection

| Framework | Detection | Skill to use |
|-----------|-----------|-------------|
| **React (Vite/CRA)** | `package.json` with `react` (but no `next`) | `/rampart-react-setup` |
| **Next.js** | Already handled as full-stack in backend detection | `/rampart-nextjs-setup` |
| **Vue** | `package.json` with `vue` | Adapt React patterns for Vue |
| **Angular** | `angular.json` exists | Adapt React patterns for Angular |
| **Vanilla JS/TS** | None of the above frameworks detected | Use `@rampart/web` adapter patterns |

## Execution Plan

### Step 1: Detect the stack

```
Scan for:
- package.json, go.mod, pom.xml, build.gradle, requirements.txt, pyproject.toml
- Framework-specific config files
- Import patterns in source files
```

Report what was found to the user before proceeding.

### Step 2: Check for Rampart

If `$ARGUMENTS` is provided, use it as the issuer URL and skip to Step 4.

Otherwise, check if Rampart is already running:

```bash
curl -sf http://localhost:8080/healthz
```

If not running, ask the user:
> "No Rampart server detected. Want me to set one up with Docker Compose?"

If yes, use the `/rampart-docker-quickstart` skill.

### Step 3: Register an OAuth client

Create a client in Rampart for this application:

- **Client ID**: Derive from `package.json` name, `go.mod` module, or project directory name
- **Type**: Public for SPAs, Confidential for server-rendered apps
- **Redirect URI**: Based on the detected dev server port (Vite=5173, CRA=3000, Next.js=3000, etc.)

### Step 4: Secure the backend

Based on detected framework, apply the appropriate skill:

- Express → `/rampart-node-setup $ARGUMENTS`
- Next.js → `/rampart-nextjs-setup $ARGUMENTS`
- FastAPI/Flask → `/rampart-python-setup $ARGUMENTS`
- Go → `/rampart-go-setup $ARGUMENTS`
- Spring Boot → `/rampart-spring-setup $ARGUMENTS`

### Step 5: Secure the frontend

If a separate frontend is detected:

- React → `/rampart-react-setup $ARGUMENTS`
- For other frameworks, adapt the React PKCE patterns

### Step 6: Add environment variables

Create or update `.env` / `.env.local` with:

```
RAMPART_ISSUER=http://localhost:8080
RAMPART_CLIENT_ID=<detected-client-id>
```

### Step 7: Verify the setup

Guide the user through testing:

1. Start Rampart (if using Docker)
2. Start the backend
3. Start the frontend
4. Navigate to a protected page
5. Verify redirect to Rampart login
6. Login and verify redirect back with user session

## Output

After completion, provide a summary:

```
Rampart Auth Setup Complete

Stack detected:
  Backend:  Express (Node.js)
  Frontend: React (Vite)

Rampart:    http://localhost:8080
Client ID:  my-app
Redirect:   http://localhost:5173/callback

Files modified:
  - src/middleware/auth.ts (JWT validation)
  - src/auth/AuthProvider.tsx (OAuth context)
  - src/auth/Callback.tsx (Token exchange)
  - .env (Rampart config)

Test it:
  1. docker compose up -d     # Start Rampart
  2. npm run dev               # Start your app
  3. Open http://localhost:5173/dashboard
  4. You'll be redirected to Rampart login
```

## Supported Stack Combinations

| Backend | Frontend | Status |
|---------|----------|--------|
| Express | React | Full support |
| Express | Next.js | Full support |
| Next.js (full-stack) | — | Full support |
| FastAPI | React | Full support |
| FastAPI | Vue/Angular | Adapt React patterns |
| Flask | React | Full support |
| Go (chi/gin/fiber) | React | Full support |
| Spring Boot | React | Full support |
| Spring Boot | Angular | Adapt React patterns |

## Checklist

- [ ] Tech stack detected and confirmed with user
- [ ] Rampart server running (new or existing)
- [ ] OAuth client registered
- [ ] Backend auth middleware installed
- [ ] Frontend auth flow configured
- [ ] Environment variables set
- [ ] End-to-end login flow tested
