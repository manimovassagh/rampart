---
sidebar_position: 5
title: Go
description: Integrate Rampart authentication into Go applications with middleware for net/http, chi, gin, and fiber.
---

# Go Adapter

The `rampart-go` adapter provides middleware for protecting Go HTTP services with Rampart-issued JWT tokens. It supports the standard library `net/http`, as well as popular routers including chi, gin, and fiber.

## Installation

```bash
go get github.com/manimovassagh/rampart-go
```

## Quick Start (net/http)

```go
package main

import (
	"encoding/json"
	"log"
	"net/http"

	rampart "github.com/manimovassagh/rampart-go"
)

func main() {
	auth, err := rampart.New(rampart.Config{
		IssuerURL: "https://auth.example.com",
		Audience:  "my-api",
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	// Public route
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Protected route
	mux.Handle("GET /api/profile", auth.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := rampart.ClaimsFromContext(r.Context())
		json.NewEncoder(w).Encode(map[string]any{
			"userId": claims.Subject,
			"email":  claims.Email,
			"roles":  claims.Roles,
		})
	})))

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## Configuration

```go
auth, err := rampart.New(rampart.Config{
	// Required
	IssuerURL: "https://auth.example.com",
	Audience:  "my-api",

	// Optional
	Realm:          "default",       // Organization/realm
	ClockTolerance: 5 * time.Second, // Clock skew tolerance (default: 5s)
	JWKSCacheTTL:   10 * time.Minute,// JWKS cache duration (default: 10m)
	RequiredClaims: []string{"email"},// Claims that must exist in token
})
```

### Configuration from Environment

```go
auth, err := rampart.NewFromEnv()
```

Reads `RAMPART_URL`, `RAMPART_CLIENT_ID`, and `RAMPART_REALM` from the environment.

## Claims

The `Claims` struct provides typed access to token claims:

```go
type Claims struct {
	Subject   string   `json:"sub"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
	Scope     string   `json:"scope"`
	OrgID     string   `json:"org_id"`
	Issuer    string   `json:"iss"`
	Audience  []string `json:"aud"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
}

// Helper methods
claims.HasRole("admin")                    // bool
claims.HasAnyRole("admin", "editor")       // bool
claims.HasScope("users:read")              // bool
claims.HasAllScopes("users:read", "users:write") // bool
```

Extract claims from the request context:

```go
claims := rampart.ClaimsFromContext(r.Context())
if claims == nil {
	// No authenticated user
}
```

## Middleware

### `RequireAuth`

Verifies the bearer token. Returns 401 if missing or invalid.

```go
mux.Handle("GET /api/data", auth.RequireAuth(handler))
```

### `RequireRoles`

Requires all specified roles. Returns 403 if any role is missing.

```go
mux.Handle("DELETE /api/admin/users/{id}",
	auth.RequireAuth(
		auth.RequireRoles("admin")(handler),
	),
)
```

### `RequireScopes`

Requires all specified scopes.

```go
mux.Handle("POST /api/emails",
	auth.RequireAuth(
		auth.RequireScopes("email:send")(handler),
	),
)
```

### `OptionalAuth`

Attempts to verify the token if present but allows unauthenticated requests through. `ClaimsFromContext` returns `nil` for unauthenticated requests.

```go
mux.Handle("GET /api/feed", auth.OptionalAuth(handler))
```

## Router Integration: chi

```go
package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	rampart "github.com/manimovassagh/rampart-go"
	rampartchi "github.com/manimovassagh/rampart-go/chi"
)

func main() {
	auth, err := rampart.New(rampart.Config{
		IssuerURL: "https://auth.example.com",
		Audience:  "my-api",
	})
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// Public routes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(rampartchi.Middleware(auth))

		r.Get("/api/profile", func(w http.ResponseWriter, r *http.Request) {
			claims := rampart.ClaimsFromContext(r.Context())
			json.NewEncoder(w).Encode(map[string]any{
				"userId": claims.Subject,
				"email":  claims.Email,
			})
		})

		r.Get("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
			claims := rampart.ClaimsFromContext(r.Context())
			json.NewEncoder(w).Encode(map[string]any{
				"tasks":  []string{"Review PR", "Deploy"},
				"userId": claims.Subject,
			})
		})
	})

	// Admin routes — require admin role
	r.Group(func(r chi.Router) {
		r.Use(rampartchi.Middleware(auth))
		r.Use(rampartchi.RequireRoles(auth, "admin"))

		r.Get("/api/admin/stats", func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{
				"totalUsers": 1234,
			})
		})
	})

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
```

## Router Integration: Gin

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	rampart "github.com/manimovassagh/rampart-go"
	rampartgin "github.com/manimovassagh/rampart-go/gin"
)

func main() {
	auth, err := rampart.New(rampart.Config{
		IssuerURL: "https://auth.example.com",
		Audience:  "my-api",
	})
	if err != nil {
		panic(err)
	}

	r := gin.Default()

	// Public route
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Protected routes
	api := r.Group("/api")
	api.Use(rampartgin.Middleware(auth))
	{
		api.GET("/profile", func(c *gin.Context) {
			claims := rampartgin.ClaimsFromContext(c)
			c.JSON(http.StatusOK, gin.H{
				"userId": claims.Subject,
				"email":  claims.Email,
				"roles":  claims.Roles,
			})
		})

		api.GET("/tasks", func(c *gin.Context) {
			claims := rampartgin.ClaimsFromContext(c)
			c.JSON(http.StatusOK, gin.H{
				"tasks":  []string{"Review PR", "Deploy"},
				"userId": claims.Subject,
			})
		})
	}

	// Admin routes
	admin := r.Group("/api/admin")
	admin.Use(rampartgin.Middleware(auth))
	admin.Use(rampartgin.RequireRoles(auth, "admin"))
	{
		admin.GET("/stats", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"totalUsers": 1234})
		})
	}

	r.Run(":8080")
}
```

## Router Integration: Fiber

```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	rampart "github.com/manimovassagh/rampart-go"
	rampartfiber "github.com/manimovassagh/rampart-go/fiber"
)

func main() {
	auth, err := rampart.New(rampart.Config{
		IssuerURL: "https://auth.example.com",
		Audience:  "my-api",
	})
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	// Public route
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Protected routes
	api := app.Group("/api", rampartfiber.Middleware(auth))

	api.Get("/profile", func(c *fiber.Ctx) error {
		claims := rampartfiber.ClaimsFromContext(c)
		return c.JSON(fiber.Map{
			"userId": claims.Subject,
			"email":  claims.Email,
			"roles":  claims.Roles,
		})
	})

	api.Get("/tasks", func(c *fiber.Ctx) error {
		claims := rampartfiber.ClaimsFromContext(c)
		return c.JSON(fiber.Map{
			"tasks":  []string{"Review PR", "Deploy"},
			"userId": claims.Subject,
		})
	})

	// Admin routes
	admin := app.Group("/api/admin",
		rampartfiber.Middleware(auth),
		rampartfiber.RequireRoles(auth, "admin"),
	)

	admin.Get("/stats", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"totalUsers": 1234})
	})

	log.Fatal(app.Listen(":8080"))
}
```

## JWKS Verification

The adapter handles JWKS fetching and caching automatically:

1. On first request, fetches the JWKS from `{IssuerURL}/.well-known/jwks.json`
2. Caches the key set for `JWKSCacheTTL` (default: 10 minutes)
3. If a token references an unknown `kid`, forces a JWKS refresh
4. Verifies RS256/RS384/RS512 and ES256/ES384/ES512 signatures

### Manual Token Verification

If you need to verify tokens outside of HTTP middleware:

```go
claims, err := auth.VerifyToken(ctx, tokenString)
if err != nil {
	switch {
	case errors.Is(err, rampart.ErrTokenExpired):
		// Token has expired
	case errors.Is(err, rampart.ErrInvalidSignature):
		// Token signature is invalid
	case errors.Is(err, rampart.ErrInvalidClaims):
		// Required claims are missing
	default:
		// Other verification error
	}
}
```

## Custom Error Responses

Override the default 401/403 responses:

```go
auth, err := rampart.New(rampart.Config{
	IssuerURL: "https://auth.example.com",
	Audience:  "my-api",
	OnUnauthorized: func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "unauthorized",
			"message": "Please provide a valid access token.",
		})
	},
	OnForbidden: func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "forbidden",
			"message": "You do not have permission to access this resource.",
		})
	},
})
```

## Client Credentials Flow

For service-to-service communication:

```go
client, err := rampart.NewClient(rampart.ClientConfig{
	IssuerURL:    "https://auth.example.com",
	ClientID:     "my-service",
	ClientSecret: os.Getenv("RAMPART_CLIENT_SECRET"),
})
if err != nil {
	log.Fatal(err)
}

// Get a token using client credentials
token, err := client.ClientCredentialsToken(ctx, []string{"users:read"})
if err != nil {
	log.Fatal(err)
}

// Use the token
req, _ := http.NewRequest("GET", "https://api.internal/users", nil)
req.Header.Set("Authorization", "Bearer "+token.AccessToken)
resp, err := http.DefaultClient.Do(req)
```

The client caches tokens and refreshes them automatically before expiry.
