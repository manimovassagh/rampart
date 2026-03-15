# Vanilla Web Frontend Sample

A vanilla TypeScript single-page application that demonstrates the [@rampart-auth/web](../../adapters/frontend/web/) adapter with OAuth 2.0 PKCE authentication. No framework required -- just HTML, CSS, and TypeScript compiled with Vite.

## Prerequisites

- Node.js 18+
- A running Rampart server (default: `http://localhost:8080`)
- A running backend on port 3001 (see [express-backend](../express-backend/), [go-backend](../go-backend/), or [fastapi-backend](../fastapi-backend/))

## Quick Start

```bash
cd cookbook/web-frontend
npm install
npm run dev
```

The app starts on `http://localhost:3000` and proxies `/api` requests to `http://localhost:3001`.

## Features

- **Login / Logout** -- OAuth 2.0 Authorization Code with PKCE via `RampartClient`
- **User Profile** -- Displays authenticated user details after login
- **API Tester** -- Call `/api/profile`, `/api/claims`, and `/me` endpoints with automatic token attachment
- **Unauthenticated Tests** -- Demonstrate 401 responses when calling protected endpoints without a token
- **Token Persistence** -- Tokens stored in `localStorage` and restored on page load

## Configuration

The Rampart issuer URL and OAuth client ID are configured at the top of `src/main.ts`. The Vite dev server port and API proxy are configured in `vite.config.ts`.
