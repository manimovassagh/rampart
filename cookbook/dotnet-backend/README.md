# Rampart .NET Backend Sample

A minimal ASP.NET Core Web API that authenticates requests using Rampart JWT tokens. This is a drop-in replacement for the Express backend sample — same port (3001), same endpoints, same JSON responses.

## Prerequisites

- [.NET 8.0 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
- A running Rampart server at `http://localhost:8080` (or set `RAMPART_ISSUER`)

## Quick Start

```bash
cd cookbook/dotnet-backend
dotnet run
```

The server starts on **http://localhost:3001**.

## Configuration

The Rampart issuer URL can be set in three ways (highest priority first):

1. `appsettings.json` — `Rampart:Issuer`
2. Environment variable — `RAMPART_ISSUER`
3. Default — `http://localhost:8080`

## Endpoints

| Method | Path                    | Auth       | Description                   |
|--------|-------------------------|------------|-------------------------------|
| GET    | `/api/health`           | Public     | Health check + issuer info    |
| GET    | `/api/profile`          | JWT        | Current user profile          |
| GET    | `/api/claims`           | JWT        | Raw JWT claims                |
| GET    | `/api/editor/dashboard` | JWT+editor | Editor dashboard (role-gated) |
| GET    | `/api/manager/reports`  | JWT+manager| Manager reports (role-gated)  |

## Error Responses

All error responses follow the same JSON format as the Express backend:

```json
// 401 Unauthorized
{"error":"unauthorized","error_description":"Missing authorization header.","status":401}

// 403 Forbidden
{"error":"forbidden","error_description":"Missing required role(s): editor","status":403}
```

## Using with the React App

Point the React sample app at this backend:

```bash
cd cookbook/react-app
VITE_API_URL=http://localhost:3001 npm run dev
```
