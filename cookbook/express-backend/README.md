# Express Backend Sample

A Node.js/Express backend that demonstrates the [@rampart-auth/node](../../adapters/backend/node/) adapter for JWT authentication and RBAC. This serves as the reference backend implementation for the React and web frontend samples.

## Prerequisites

- Node.js 18+
- A running Rampart server (default: `http://localhost:8080`)

## Quick Start

```bash
cd cookbook/express-backend
npm install
npm run dev
```

The server starts on `http://localhost:3001`.

## Configuration

| Environment Variable | Default                 | Description                    |
|----------------------|-------------------------|--------------------------------|
| `RAMPART_ISSUER`     | `http://localhost:8080` | Rampart server URL             |
| `PORT`               | `3001`                  | Port to listen on              |

## Endpoints

| Method | Path                    | Auth     | Description                      |
|--------|-------------------------|----------|----------------------------------|
| GET    | `/api/health`           | Public   | Health check, returns issuer URL |
| GET    | `/api/profile`          | JWT      | Returns authenticated user info  |
| GET    | `/api/claims`           | JWT      | Returns all JWT claims           |
| GET    | `/api/editor/dashboard` | JWT+RBAC | Requires "editor" role           |
| GET    | `/api/manager/reports`  | JWT+RBAC | Requires "manager" role          |

## Usage with the React Frontend

Start both the backend and the React sample app:

```bash
# Terminal 1 — backend
cd cookbook/express-backend
npm run dev

# Terminal 2 — frontend
cd cookbook/react-app
npm run dev
```

The React app on port 3002 proxies `/api` requests to `http://localhost:3001`.
