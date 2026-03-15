# React Sample App

A React 19 single-page application that demonstrates the [@rampart-auth/react](../../adapters/frontend/react/) adapter with OAuth 2.0 PKCE authentication. Uses React Router for navigation and Tailwind CSS for styling.

## Prerequisites

- Node.js 18+
- A running Rampart server (default: `http://localhost:8080`)
- A running backend on port 3001 (see [express-backend](../express-backend/), [go-backend](../go-backend/), or [fastapi-backend](../fastapi-backend/))

## Quick Start

```bash
cd cookbook/react-app
npm install
npm run dev
```

The app starts on `http://localhost:3002` and proxies `/api` requests to `http://localhost:3001`.

## Features

- **Login / Logout** -- OAuth 2.0 Authorization Code with PKCE via Rampart
- **Dashboard** -- Displays authenticated user profile, JWT claims, and an API tester for backend endpoints
- **API Tester** -- Call protected backend endpoints with the user's access token
- **RBAC** -- Role-based access control with `ProtectedRoute` components
- **Admin Page** -- Restricted to users with the "admin" role

## Pages

| Path         | Description                                      |
|--------------|--------------------------------------------------|
| `/`          | Landing page (redirects to dashboard if logged in) |
| `/callback`  | OAuth callback handler                           |
| `/dashboard` | Protected user dashboard with API tester         |
| `/admin`     | Protected admin page (requires "admin" role)     |

## Configuration

The Rampart issuer URL and OAuth client ID are configured in `src/main.tsx`. The Vite dev server port and API proxy are configured in `vite.config.ts`.
