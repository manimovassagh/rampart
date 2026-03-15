# Go Backend Sample

A sample Go backend that demonstrates the Rampart Go adapter for JWT authentication and RBAC. This is a drop-in replacement for the Express backend sample — same port (3001), same endpoints, same JSON responses.

## Prerequisites

- Go 1.23+
- A running Rampart server (default: `http://localhost:8080`)

## Quick Start

```bash
cd cookbook/go-backend
go run main.go
```

The server starts on `http://localhost:3001`.

## Configuration

| Environment Variable | Default                  | Description                |
|---------------------|--------------------------|----------------------------|
| `RAMPART_ISSUER`    | `http://localhost:8080`  | Rampart server URL         |
| `PORT`              | `3001`                   | Port to listen on          |

## Endpoints

| Method | Path                    | Auth     | Description                      |
|--------|-------------------------|----------|----------------------------------|
| GET    | `/api/health`           | Public   | Health check, returns issuer URL |
| GET    | `/api/profile`          | JWT      | Returns authenticated user info  |
| GET    | `/api/claims`           | JWT      | Returns all JWT claims           |
| GET    | `/api/editor/dashboard` | JWT+RBAC | Requires "editor" role           |
| GET    | `/api/manager/reports`  | JWT+RBAC | Requires "manager" role          |

## Usage with the React Frontend

This backend is fully compatible with the React sample app. Start both:

```bash
# Terminal 1 — backend
cd cookbook/go-backend
go run main.go

# Terminal 2 — frontend
cd cookbook/react-app
npm run dev
```

The React app talks to `http://localhost:3001` by default.
