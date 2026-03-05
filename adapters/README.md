# Rampart Adapters

Official SDK adapters for integrating with a Rampart IAM server.

## Backend

Server-side middleware for verifying Rampart JWTs on protected API routes.

| Adapter | Package | Framework | Status |
|---------|---------|-----------|--------|
| [Node.js](./backend/node/) | `@rampart/node` | Express >=4 | Ready |
| [Go](./backend/go/) | `rampart` (Go module) | net/http, chi, gin | Ready |
| [Python](./backend/python/) | `rampart-python` | FastAPI, Flask | Ready |
| [Spring Boot](./backend/spring/) | `rampart-spring-boot-starter` | Spring Boot 3.x | Ready |

## Frontend

Client-side libraries for login, token management, and authenticated API calls.

| Adapter | Package | Framework | Status |
|---------|---------|-----------|--------|
| [Web](./frontend/web/) | `@rampart/web` | Any (vanilla JS/TS) | Ready |
| [React](./frontend/react/) | `@rampart/react` | React >=18 | Ready |
| [Next.js](./frontend/nextjs/) | `@rampart/nextjs` | Next.js 13+ | Ready |
