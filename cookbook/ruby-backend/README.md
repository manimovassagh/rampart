# Ruby (Sinatra) Backend — Rampart Cookbook

A minimal Sinatra API that validates Rampart-issued JWTs using the JWKS endpoint.
Mirrors the same five endpoints as every other cookbook backend.

## Prerequisites

- Ruby 3.x
- Bundler (`gem install bundler`)
- A running Rampart server on `http://localhost:8080`

## Setup

```bash
bundle install
ruby app.rb          # starts on http://localhost:3001
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
