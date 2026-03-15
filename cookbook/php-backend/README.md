# PHP (Slim 4) Backend — Rampart Cookbook

A minimal Slim 4 API that validates Rampart-issued JWTs using the JWKS endpoint.
Mirrors the same five endpoints as every other cookbook backend.

## Prerequisites

- PHP 8.1+
- Composer (`https://getcomposer.org`)
- A running Rampart server on `http://localhost:8080`

## Setup

```bash
composer install
php -S 0.0.0.0:3001 -t public   # starts on http://localhost:3001
```

## Endpoints

| Route                      | Auth       | Description            |
|----------------------------|------------|------------------------|
| `GET /api/health`          | public     | Health check           |
| `GET /api/profile`         | JWT        | Authenticated profile  |
| `GET /api/claims`          | JWT        | Raw token claims       |
| `GET /api/editor/dashboard`| JWT+editor | Editor RBAC dashboard  |
| `GET /api/manager/reports` | JWT+manager| Manager RBAC reports   |

## Environment Variables

| Variable          | Default                  |
|-------------------|--------------------------|
| `PORT`            | `3001`                   |
| `RAMPART_ISSUER`  | `http://localhost:8080`  |

## Usage with the React Frontend

This backend is fully compatible with the React sample app. Start both:

```bash
# Terminal 1 — backend
cd cookbook/php-backend
php -S 0.0.0.0:3001 -t public

# Terminal 2 — frontend
cd cookbook/react-app
npm run dev
```

The React app talks to `http://localhost:3001` by default.
