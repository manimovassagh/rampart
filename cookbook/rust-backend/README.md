# Rust/Actix-web Backend — Rampart Cookbook

A sample backend using **Actix-web 4** that validates Rampart JWTs via JWKS.

## Prerequisites

- Rust 1.70+ (`rustup` recommended)
- Rampart running on `http://localhost:8080`

## Quick Start

```bash
cargo build --release
RAMPART_ISSUER=http://localhost:8080 ./target/release/rust-backend
```

## Routes

| Endpoint                  | Auth     | Description                    |
|---------------------------|----------|--------------------------------|
| `GET /api/health`         | Public   | Health check                   |
| `GET /api/profile`        | JWT      | Returns user profile           |
| `GET /api/claims`         | JWT      | Returns raw JWT claims         |
| `GET /api/editor/dashboard` | JWT + editor role | Editor dashboard    |
| `GET /api/manager/reports`  | JWT + manager role | Manager reports   |

## Configuration

| Env Variable      | Default                  | Description          |
|-------------------|--------------------------|----------------------|
| `RAMPART_ISSUER`  | `http://localhost:8080`  | Rampart server URL   |
| `PORT`            | `3001`                   | Server listen port   |
