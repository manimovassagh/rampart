---
name: rampart-node-setup
description: Set up @rampart/node authentication in an Express application. Installs the package, configures middleware, protects routes, and adds typed claims to req.auth. Use when adding Rampart auth to a Node.js/Express backend.
argument-hint: [issuer-url]
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---

# Add Rampart Authentication to an Express App

Set up the `@rampart/node` middleware so this Express app verifies JWTs from a Rampart IAM server.

## What This Skill Does

1. Installs `@rampart/node` and `jose`
2. Adds `rampartAuth()` middleware to protect routes
3. Gives you typed `req.auth` with user claims (sub, email, org_id, etc.)
4. Returns Rampart-standard 401 JSON errors on auth failure

## Step-by-Step

### 1. Install the package

```bash
npm install @rampart/node jose
```

### 2. Find the main app/server file

Look for the Express app setup — typically `app.ts`, `server.ts`, `index.ts`, or `src/app.ts`.

### 3. Add the middleware

Import and configure `rampartAuth`:

```typescript
import { rampartAuth } from "@rampart/node";

// Option A: Protect all routes globally
app.use(rampartAuth({ issuer: "$ARGUMENTS" }));

// Option B: Protect specific routes
const auth = rampartAuth({ issuer: "$ARGUMENTS" });
app.get("/api/profile", auth, (req, res) => {
  res.json({
    userId: req.auth!.sub,
    email: req.auth!.email,
    org: req.auth!.org_id,
  });
});
```

If `$ARGUMENTS` is empty, use `process.env.RAMPART_ISSUER` or `"http://localhost:8080"` as the default.

### 4. Available claims on `req.auth`

After middleware runs, `req.auth` contains:

| Claim                | Type      | Example                |
|----------------------|-----------|------------------------|
| `sub`                | `string`  | User UUID              |
| `iss`                | `string`  | Issuer URL             |
| `iat`                | `number`  | Issued-at timestamp    |
| `exp`                | `number`  | Expiry timestamp       |
| `org_id`             | `string`  | Organization UUID      |
| `preferred_username` | `string`  | Username               |
| `email`              | `string`  | Email address          |
| `email_verified`     | `boolean` | Verification status    |
| `given_name`         | `string?` | First name (optional)  |
| `family_name`        | `string?` | Last name (optional)   |

### 5. Error responses

The middleware returns 401 JSON matching Rampart's format:

```json
{ "error": "unauthorized", "error_description": "...", "status": 401 }
```

Messages:
- `"Missing authorization header."` — no Authorization header sent
- `"Invalid authorization header format."` — not Bearer scheme
- `"Invalid or expired access token."` — bad signature, wrong issuer, or expired

### 6. Frontend integration

When setting up a frontend (React, etc.) to work with a Rampart-protected backend:

- Store the access token from Rampart's `/login` response
- Attach it to every API request: `Authorization: Bearer <token>`
- Handle 401 responses by redirecting to login or refreshing the token
- Use the `/me` endpoint to fetch current user info on app load

Example fetch wrapper:

```typescript
async function apiCall(path: string) {
  const token = localStorage.getItem("rampart_token");
  const res = await fetch(path, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (res.status === 401) {
    // Token expired or invalid — redirect to login
    window.location.href = "/login";
    return;
  }
  return res.json();
}
```

## Checklist

- [ ] `@rampart/node` and `jose` installed
- [ ] `rampartAuth({ issuer })` middleware added to routes
- [ ] Protected routes use `req.auth` for user identity
- [ ] Frontend sends `Authorization: Bearer <token>` header
- [ ] 401 responses handled on the frontend (redirect to login / refresh)
