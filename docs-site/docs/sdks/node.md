---
sidebar_position: 2
title: Node.js / Express
description: Integrate Rampart authentication into Node.js and Express applications with the @rampart/node SDK adapter.
---

# Node.js / Express Adapter

The `@rampart/node` adapter provides Express middleware for protecting routes with Rampart-issued JWT tokens. It handles OIDC discovery, JWKS-based token verification, and user context extraction.

## Installation

```bash
npm install @rampart/node
```

```bash
yarn add @rampart/node
```

```bash
pnpm add @rampart/node
```

## Quick Start

```typescript
import express from "express";
import { RampartAuth } from "@rampart/node";

const app = express();

const auth = new RampartAuth({
  issuerUrl: process.env.RAMPART_URL || "https://auth.example.com",
  audience: process.env.RAMPART_CLIENT_ID || "my-api",
});

// Public route
app.get("/health", (req, res) => {
  res.json({ status: "ok" });
});

// Protected route
app.get("/api/profile", auth.requireAuth(), (req, res) => {
  res.json({
    userId: req.auth.sub,
    email: req.auth.email,
    roles: req.auth.roles,
  });
});

app.listen(3000, () => {
  console.log("Server running on http://localhost:3000");
});
```

## Configuration

```typescript
const auth = new RampartAuth({
  // Required
  issuerUrl: "https://auth.example.com",
  audience: "my-api",

  // Optional
  realm: "default",                    // Organization/realm
  jwksCache: {
    ttl: 600,                          // JWKS cache TTL in seconds (default: 600)
    refreshInterval: 300,              // Background refresh interval (default: 300)
  },
  clockTolerance: 5,                   // Seconds of clock skew tolerance (default: 5)
  requiredClaims: ["email"],           // Claims that must be present in the token
});
```

### Environment Variables

You can also configure via environment variables:

```bash
RAMPART_URL=https://auth.example.com
RAMPART_CLIENT_ID=my-api
RAMPART_REALM=default
```

```typescript
const auth = RampartAuth.fromEnv();
```

## Middleware

### `requireAuth()`

Verifies the bearer token and attaches decoded claims to `req.auth`. Returns 401 if the token is missing or invalid.

```typescript
app.get("/api/data", auth.requireAuth(), (req, res) => {
  // req.auth is guaranteed to be present
  console.log(req.auth.sub);    // User ID
  console.log(req.auth.email);  // Email
  console.log(req.auth.roles);  // Roles array
  console.log(req.auth.scope);  // Scopes string
  res.json({ data: "protected" });
});
```

### `requireRoles(...roles)`

Requires the user to have all specified roles. Returns 403 if any role is missing.

```typescript
app.delete(
  "/api/admin/users/:id",
  auth.requireAuth(),
  auth.requireRoles("admin"),
  (req, res) => {
    res.json({ deleted: req.params.id });
  }
);
```

### `requireScopes(...scopes)`

Requires the token to include all specified scopes. Returns 403 if any scope is missing.

```typescript
app.get(
  "/api/emails",
  auth.requireAuth(),
  auth.requireScopes("email", "profile"),
  (req, res) => {
    res.json({ email: req.auth.email });
  }
);
```

### `optionalAuth()`

Attempts to verify the token if present but does not reject the request if no token is provided. `req.auth` will be `null` for unauthenticated requests.

```typescript
app.get("/api/feed", auth.optionalAuth(), (req, res) => {
  if (req.auth) {
    res.json({ feed: "personalized", user: req.auth.sub });
  } else {
    res.json({ feed: "public" });
  }
});
```

## TypeScript Support

The adapter ships with full TypeScript definitions. Extend the Express `Request` type by importing the module:

```typescript
import "@rampart/node";

// req.auth is now typed on all Express Request objects
```

The `AuthContext` type contains:

```typescript
interface AuthContext {
  sub: string;           // User ID
  email?: string;        // Email address
  name?: string;         // Display name
  roles: string[];       // Assigned roles
  scope: string;         // Space-separated scopes
  orgId?: string;        // Organization ID
  iss: string;           // Issuer URL
  aud: string | string[];// Audience
  exp: number;           // Expiration timestamp
  iat: number;           // Issued-at timestamp
  raw: string;           // Raw JWT string
}
```

## Error Handling

The middleware emits standard error responses. Customize them with an error handler:

```typescript
import { RampartAuthError, TokenExpiredError } from "@rampart/node";

const auth = new RampartAuth({
  issuerUrl: "https://auth.example.com",
  audience: "my-api",
  onError: (err, req, res) => {
    if (err instanceof TokenExpiredError) {
      res.status(401).json({
        error: "token_expired",
        message: "Your session has expired. Please log in again.",
      });
    } else {
      res.status(401).json({
        error: "unauthorized",
        message: "Invalid or missing authentication token.",
      });
    }
  },
});
```

## Token Verification Details

The adapter performs the following checks on every request:

1. Extracts the token from the `Authorization: Bearer <token>` header
2. Fetches JWKS from `{issuerUrl}/.well-known/jwks.json` (cached)
3. Verifies the JWT signature using the matching key from JWKS
4. Validates standard claims: `iss`, `aud`, `exp`, `iat`
5. Applies clock tolerance for `exp` and `nbf` claims
6. Checks any `requiredClaims` specified in configuration
7. Attaches decoded claims to `req.auth`

## Full Working Example

```typescript
import express from "express";
import cors from "cors";
import { RampartAuth } from "@rampart/node";

const app = express();
app.use(cors());
app.use(express.json());

const auth = new RampartAuth({
  issuerUrl: process.env.RAMPART_URL || "https://auth.example.com",
  audience: process.env.RAMPART_CLIENT_ID || "task-api",
});

// Health check — public
app.get("/health", (_req, res) => {
  res.json({ status: "ok" });
});

// List tasks — requires authentication
app.get("/api/tasks", auth.requireAuth(), (req, res) => {
  const userId = req.auth.sub;
  // Fetch tasks for this user from your database
  res.json({
    tasks: [
      { id: "1", title: "Review PR", assignee: userId },
      { id: "2", title: "Deploy to staging", assignee: userId },
    ],
  });
});

// Create task — requires authentication and "write" scope
app.post(
  "/api/tasks",
  auth.requireAuth(),
  auth.requireScopes("tasks:write"),
  (req, res) => {
    const { title } = req.body;
    res.status(201).json({
      id: "3",
      title,
      assignee: req.auth.sub,
      createdAt: new Date().toISOString(),
    });
  }
);

// Admin endpoint — requires "admin" role
app.get(
  "/api/admin/stats",
  auth.requireAuth(),
  auth.requireRoles("admin"),
  (_req, res) => {
    res.json({
      totalUsers: 1234,
      activeToday: 567,
      tasksCreated: 8901,
    });
  }
);

// Global error handler
app.use((err: Error, _req: express.Request, res: express.Response, _next: express.NextFunction) => {
  console.error("Unhandled error:", err.message);
  res.status(500).json({ error: "internal_error" });
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Task API running on http://localhost:${PORT}`);
});
```

## Confidential Client (Backend-to-Backend)

For server-to-server communication where your service acts as a confidential client:

```typescript
import { RampartClient } from "@rampart/node";

const client = new RampartClient({
  issuerUrl: "https://auth.example.com",
  clientId: "my-service",
  clientSecret: process.env.RAMPART_CLIENT_SECRET!,
});

// Get a token using client credentials flow
const token = await client.getClientCredentialsToken({
  scopes: ["users:read", "users:write"],
});

// Use the token to call another service
const response = await fetch("https://api.internal/users", {
  headers: {
    Authorization: `Bearer ${token.accessToken}`,
  },
});
```

The client automatically handles token caching and renewal.
