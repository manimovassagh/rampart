# Rampart Sample Applications

Working examples that show how to integrate with a Rampart IAM server using the official adapters. Every backend sample exposes the same API (same endpoints, same JSON shapes, same port) so any frontend can be paired with any backend.

## Sample Index

### Backend Samples

All backend samples listen on port **3001** and expose the same routes:

| Sample | Tech Stack | Adapter | How to Run |
|--------|-----------|---------|------------|
| [express-backend](./express-backend/) | Node.js, Express 5, TypeScript | [`@rampart-auth/node`](../adapters/backend/node/) | `npm install && npm run dev` |
| [go-backend](./go-backend/) | Go 1.23+, net/http | [`rampart` Go module](../adapters/backend/go/) | `go run main.go` |
| [fastapi-backend](./fastapi-backend/) | Python 3.9+, FastAPI, Uvicorn | [`rampart-python`](../adapters/backend/python/) | `pip install -r requirements.txt && python main.py` |
| [spring-backend](./spring-backend/) | Java 17+, Spring Boot 3.3, Spring Security | [`rampart-spring-boot-starter`](../adapters/backend/spring/) | `./mvnw spring-boot:run` |

### Frontend Samples

| Sample | Tech Stack | Adapter | Port | How to Run |
|--------|-----------|---------|------|------------|
| [react-app](./react-app/) | React 19, React Router 7, Vite, Tailwind CSS | [`@rampart-auth/react`](../adapters/frontend/react/) | 3002 | `npm install && npm run dev` |
| [web-frontend](./web-frontend/) | Vanilla TypeScript, Vite | [`@rampart-auth/web`](../adapters/frontend/web/) | 3000 | `npm install && npm run dev` |

### Shared API Surface

Every backend implements these endpoints:

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/health` | Public | Health check, returns issuer URL |
| GET | `/api/profile` | JWT required | Authenticated user profile |
| GET | `/api/claims` | JWT required | All raw JWT claims |
| GET | `/api/editor/dashboard` | `editor` role | Editor dashboard (mock data) |
| GET | `/api/manager/reports` | `manager` role | Manager reports (mock data) |

---

## Quick Start

### Prerequisites

- A running Rampart server on `http://localhost:8080` (with PostgreSQL)
- A registered OAuth client in Rampart for the frontend you plan to use

### Pick a Backend + Frontend Combo

Since every backend exposes the same API on the same port, you can mix and match freely. Open two terminals and start one backend and one frontend.

**Terminal 1 -- Start a backend (choose one):**

```bash
# Option A: Node.js / Express
cd samples/express-backend
npm install
npm run dev

# Option B: Go
cd samples/go-backend
go run main.go

# Option C: Python / FastAPI
cd samples/fastapi-backend
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
pip install -e ../../adapters/backend/python
python main.py

# Option D: Java / Spring Boot
cd samples/spring-backend
./mvnw spring-boot:run
```

**Terminal 2 -- Start a frontend (choose one):**

```bash
# Option A: React SPA (port 3002)
cd samples/react-app
npm install
npm run dev

# Option B: Vanilla JS (port 3000)
cd samples/web-frontend
npm install
npm run dev
```

**Terminal 3 -- Try it:**

1. Open the frontend URL in your browser (`http://localhost:3002` for React, `http://localhost:3000` for web).
2. Log in with your Rampart credentials.
3. Use the UI to call protected endpoints -- the frontend sends the JWT as a Bearer token and the backend verifies it via the JWKS endpoint.

### Configuration

All samples accept these environment variables:

| Variable | Default | Used By |
|----------|---------|---------|
| `RAMPART_ISSUER` | `http://localhost:8080` | All backends |
| `PORT` | `3001` | All backends |

---

## Architecture

```
Frontend (port 3000 or 3002)     Backend (port 3001)              Rampart (port 8080)
+--------------------+           +--------------------+           +--------------------+
|                    |           |                    |           |                    |
|  Login / OAuth     |-- auth -->|                    |           |  POST /login       |
|  redirect flow     |<- token --|                    |           |  POST /token       |
|                    |           |  JWT middleware     |-- JWKS -->|  GET /.well-known/ |
|  authFetch()       |-- Bearer->|  verifies token    |           |      jwks.json     |
|                    |<- JSON ---|  extracts claims   |           |                    |
+--------------------+           +--------------------+           +--------------------+
```

The frontend handles the OAuth flow directly with Rampart. API calls go through the backend, which verifies the JWT against Rampart's JWKS endpoint and extracts claims for authorization.

---

## Test Matrix

These frontend/backend combinations have been end-to-end verified against a local Rampart server:

| Frontend | Backend | Status |
|----------|---------|--------|
| React (`react-app`) | Express / Node.js (`express-backend`) | Verified |
| React (`react-app`) | Go (`go-backend`) | Verified |
| React (`react-app`) | FastAPI / Python (`fastapi-backend`) | Verified |
| React (`react-app`) | Spring Boot / Java (`spring-backend`) | Verified |
| Vanilla JS (`web-frontend`) | Express / Node.js (`express-backend`) | Verified |

All combinations that are not listed above are expected to work (the API contract is identical) but have not been formally tested yet.

---

## Adapter Documentation

For full API documentation, configuration options, and advanced usage of each adapter, see the adapter READMEs:

### Backend Adapters

- **Node.js** -- [`adapters/backend/node/README.md`](../adapters/backend/node/README.md)
  Package: `@rampart-auth/node` | Express middleware for JWT verification and RBAC

- **Go** -- [`adapters/backend/go/README.md`](../adapters/backend/go/README.md)
  Package: `rampart` (Go module) | net/http middleware, also works with chi and gin

- **Python** -- [`adapters/backend/python/README.md`](../adapters/backend/python/README.md)
  Package: `rampart-python` | FastAPI dependency and Flask decorator for JWT verification

- **Spring Boot** -- [`adapters/backend/spring/README.md`](../adapters/backend/spring/README.md)
  Package: `rampart-spring-boot-starter` | Spring Security OAuth2 Resource Server integration

### Frontend Adapters

- **React** -- [`adapters/frontend/react/README.md`](../adapters/frontend/react/README.md)
  Package: `@rampart-auth/react` | React hooks and components for auth flows

- **Web (Vanilla JS)** -- [`adapters/frontend/web/README.md`](../adapters/frontend/web/README.md)
  Package: `@rampart-auth/web` | Framework-agnostic browser SDK for login, logout, and token management

- **Next.js** -- [`adapters/frontend/nextjs/README.md`](../adapters/frontend/nextjs/README.md)
  Package: `@rampart-auth/nextjs` | Next.js 13+ App Router integration (no sample app yet)

---

## Adding a New Sample

To add a new backend sample that works with the existing frontends:

1. Listen on port `3001` (or respect the `PORT` env var).
2. Implement the five endpoints listed in the Shared API Surface table above.
3. Return the same JSON response shapes as the existing backends.
4. Enable CORS with `Access-Control-Allow-Origin: *` for local development.
5. Add a `README.md` in the sample directory with prerequisites, setup, and run instructions.
