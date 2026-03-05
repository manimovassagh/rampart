---
name: rampart-go-setup
description: Add Rampart authentication to a Go backend (net/http, chi, gin, or fiber). Sets up JWT middleware that verifies tokens against Rampart's JWKS endpoint and provides typed claims. Use when securing a Go API with Rampart.
argument-hint: [issuer-url]
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---

# Add Rampart Authentication to a Go Backend

Set up JWT-based middleware that validates Rampart access tokens via JWKS.

## What This Skill Does

1. Installs `github.com/golang-jwt/jwt/v5` and `github.com/MicahParks/keyfunc/v2`
2. Creates auth middleware with JWKS-based token validation
3. Provides typed claims in request context
4. Works with net/http, chi, gin, or fiber

## Step-by-Step

### 1. Detect the router

Check the project's imports for: `net/http`, `github.com/go-chi/chi`, `github.com/gin-gonic/gin`, or `github.com/gofiber/fiber`. Proceed with the matching setup.

### 2. Install dependencies

```bash
go get github.com/golang-jwt/jwt/v5
go get github.com/MicahParks/keyfunc/v2
```

### 3. Create the auth package

Create `internal/auth/rampart.go` (or `pkg/auth/rampart.go`):

```go
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	PreferredUsername  string `json:"preferred_username"`
	OrgID             string `json:"org_id"`
	EmailVerified     bool   `json:"email_verified"`
	GivenName         string `json:"given_name,omitempty"`
	FamilyName        string `json:"family_name,omitempty"`
	jwt.RegisteredClaims
}

type contextKey struct{}

var claimsKey = contextKey{}

type Middleware struct {
	jwks   *keyfunc.JWKS
	issuer string
}

func NewMiddleware(issuer string) (*Middleware, error) {
	jwksURL := issuer + "/.well-known/jwks.json"
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{
		RefreshInterval: 1 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	return &Middleware{jwks: jwks, issuer: issuer}, nil
}

func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, 401, "Missing authorization header.")
			return
		}

		tokenStr := authHeader[7:]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, m.jwks.Keyfunc,
			jwt.WithIssuer(m.issuer),
			jwt.WithValidMethods([]string{"RS256"}),
		)
		if err != nil || !token.Valid {
			writeError(w, 401, "Invalid or expired access token.")
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetClaims(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey).(*Claims)
	return claims
}

func writeError(w http.ResponseWriter, status int, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error":             "unauthorized",
		"error_description": desc,
		"status":            status,
	})
}
```

### 4. Wire it up

#### net/http or chi:

```go
issuer := os.Getenv("RAMPART_ISSUER")
if issuer == "" {
    issuer = "http://localhost:8080"
}

authMW, err := auth.NewMiddleware(issuer)
if err != nil {
    log.Fatal(err)
}

// Protect all routes
mux.Handle("/api/", authMW.Handler(apiRouter))

// Or protect specific routes (chi)
r.Group(func(r chi.Router) {
    r.Use(authMW.Handler)
    r.Get("/api/profile", profileHandler)
})
```

#### Access claims in a handler:

```go
func profileHandler(w http.ResponseWriter, r *http.Request) {
    claims := auth.GetClaims(r.Context())
    json.NewEncoder(w).Encode(map[string]string{
        "id":       claims.Sub,
        "email":    claims.Email,
        "username": claims.PreferredUsername,
        "org":      claims.OrgID,
    })
}
```

### 5. Environment variables

```bash
RAMPART_ISSUER=http://localhost:8080
```

If `$ARGUMENTS` is provided, use it as the issuer URL.

## Checklist

- [ ] JWT and JWKS dependencies installed
- [ ] Auth middleware created with JWKS verification
- [ ] Middleware applied to protected routes
- [ ] Handlers access claims via `auth.GetClaims(ctx)`
- [ ] Issuer URL configured via environment variable
- [ ] 401 errors return Rampart-compatible JSON
