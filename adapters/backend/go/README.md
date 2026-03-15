# Rampart Go Middleware

JWT verification middleware for Go HTTP servers. Works with `net/http`, chi, and any router that uses `http.Handler`.

## Install

```bash
go get github.com/manimovassagh/rampart/adapters/backend/go
```

## Usage

```go
package main

import (
	"fmt"
	"net/http"

	rampart "github.com/manimovassagh/rampart/adapters/backend/go"
)

func main() {
	auth := rampart.NewMiddleware(rampart.Config{
		Issuer: "https://auth.example.com",
	})

	mux := http.NewServeMux()

	// Protected route
	mux.Handle("/api/profile", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := rampart.ClaimsFromContext(r.Context())
		fmt.Fprintf(w, "Hello, %s", claims.PreferredUsername)
	})))

	// Admin-only route (chain auth + role check)
	adminOnly := rampart.RequireRoles("admin")
	mux.Handle("/api/admin", auth(adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("admin area"))
	}))))

	http.ListenAndServe(":8080", mux)
}
```

## With chi router

```go
r := chi.NewRouter()
r.Use(rampart.NewMiddleware(rampart.Config{Issuer: "https://auth.example.com"}))
r.Get("/protected", handler)
```

## API

- **`NewMiddleware(cfg Config) func(http.Handler) http.Handler`** — JWT verification middleware
- **`ClaimsFromContext(ctx context.Context) (*Claims, bool)`** — extract verified claims from context
- **`RequireRoles(roles ...string) func(http.Handler) http.Handler`** — role-based access control (use after auth middleware)

## Claims

Available via `ClaimsFromContext(r.Context())` after successful verification:

| Field              | Type       | Description                 |
|--------------------|------------|-----------------------------|
| `Sub`              | `string`   | User ID (UUID)              |
| `Iss`              | `string`   | Issuer URL                  |
| `Iat`              | `float64`  | Issued at (Unix timestamp)  |
| `Exp`              | `float64`  | Expires at (Unix timestamp) |
| `OrgID`            | `string`   | Organization ID (UUID)      |
| `PreferredUsername` | `string`  | Username                    |
| `Email`            | `string`   | Email address               |
| `EmailVerified`    | `bool`     | Whether email is verified   |
| `GivenName`        | `string`   | First name (omitempty)      |
| `FamilyName`       | `string`   | Last name (omitempty)       |
| `Roles`            | `[]string` | Assigned roles (omitempty)  |

## Error Responses

On failure the middleware returns a JSON response matching Rampart's error format:

**401 Unauthorized** — returned by `NewMiddleware`:

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
- `"Failed to fetch JWKS."` — could not retrieve the key set from the issuer
- `"Invalid or expired access token."` — signature, issuer, or expiry check failed

**403 Forbidden** — returned by `RequireRoles`:

```json
{
  "error": "forbidden",
  "error_description": "Missing required role(s): admin",
  "status": 403
}
```
