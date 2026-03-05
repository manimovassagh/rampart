# Rampart Samples

Working examples that show how to integrate with a Rampart IAM server using the official adapters.

## Available Samples

| Sample | Description | Adapter Used | Port |
|--------|-------------|-------------|------|
| [express-backend](./express-backend/) | Express API with protected routes | `@rampart/node` | 3001 |
| [web-frontend](./web-frontend/) | Browser app with login/logout UI | `@rampart/web` | 3000 |
| [react-app](./react-app/) | React SPA with routing, auth, RBAC | `@rampart/react` | 3002 |

## Quick Start

**Prerequisites:** Rampart server running on `http://localhost:8080` (with PostgreSQL).

### 1. Start the backend

```bash
cd samples/express-backend
npm install
npm run dev
# → http://localhost:3001
```

### 2. Start the frontend

```bash
cd samples/web-frontend
npm install
npm run dev
# → http://localhost:3000
```

### 3. Try it

1. Open http://localhost:3000
2. Login with your Rampart credentials
3. Click "Fetch /api/profile" to call the protected Express endpoint
4. The Express backend verifies the JWT via `@rampart/node` and returns your claims

## Architecture

```
Browser (port 3000)          Express (port 3001)          Rampart (port 8080)
┌──────────────────┐         ┌──────────────────┐         ┌──────────────────┐
│  @rampart/web    │─login──▶│                  │         │  POST /login     │
│                  │◀─token──│                  │         │  GET /me         │
│                  │         │  @rampart/node   │─JWKS───▶│  GET /.well-     │
│  authFetch()     │─Bearer─▶│  verifies JWT    │         │    known/jwks    │
│                  │◀─data───│  req.auth.sub    │         │                  │
└──────────────────┘         └──────────────────┘         └──────────────────┘
```

Note: The frontend talks to Rampart directly for login (no backend proxy needed). API calls go through the Express backend which verifies the JWT.
