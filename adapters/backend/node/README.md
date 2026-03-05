# @rampart/node

Express middleware for verifying [Rampart](https://github.com/manimovassagh/rampart) JWTs. Handles JWKS fetching, RS256 verification, and claim extraction with zero configuration beyond the issuer URL.

## Install

```bash
npm install @rampart/node jose
```

`express` (>=4) is a peer dependency.

## Quick Start

```typescript
import express from "express";
import { rampartAuth } from "@rampart/node";

const app = express();

// Protect all routes
app.use(rampartAuth({ issuer: "http://localhost:8080" }));

app.get("/api/me", (req, res) => {
  res.json({ userId: req.auth!.sub, org: req.auth!.org_id });
});
```

### Per-route protection

```typescript
const auth = rampartAuth({ issuer: "http://localhost:8080" });

app.get("/public", (_req, res) => res.json({ ok: true }));
app.get("/api/data", auth, (req, res) => {
  res.json({ user: req.auth!.preferred_username });
});
```

## API

### `rampartAuth(config: RampartConfig)`

Returns an Express middleware that:

1. Extracts the Bearer token from the `Authorization` header
2. Fetches the JWKS from `{issuer}/.well-known/jwks.json` (cached automatically)
3. Verifies the RS256 signature, issuer, and expiry
4. Sets `req.auth` with typed claims

#### Config

| Property | Type     | Description                       |
|----------|----------|-----------------------------------|
| `issuer` | `string` | Rampart server URL (e.g. `http://localhost:8080`) |

### `RampartClaims`

Available on `req.auth` after successful verification:

| Field                | Type      | Description                |
|----------------------|-----------|----------------------------|
| `iss`                | `string`  | Issuer URL                 |
| `sub`                | `string`  | User ID (UUID)             |
| `iat`                | `number`  | Issued at (Unix timestamp) |
| `exp`                | `number`  | Expires at (Unix timestamp)|
| `org_id`             | `string`  | Organization ID (UUID)     |
| `preferred_username` | `string`  | Username                   |
| `email`              | `string`  | Email address              |
| `email_verified`     | `boolean` | Whether email is verified  |
| `given_name`         | `string?` | First name (if set)        |
| `family_name`        | `string?` | Last name (if set)         |

### Error Responses

On failure the middleware returns a `401` JSON response matching Rampart's error format:

```json
{
  "error": "unauthorized",
  "error_description": "Missing authorization header.",
  "status": 401
}
```

Error messages:
- `"Missing authorization header."` — no `Authorization` header
- `"Invalid authorization header format."` — not a `Bearer` token
- `"Invalid or expired access token."` — signature, issuer, or expiry check failed

## TypeScript

The package ships with full type definitions. `req.auth` is typed via Express module augmentation — no extra setup needed.

## License

MIT
