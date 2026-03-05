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
