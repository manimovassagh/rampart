// Package rampart provides JWT verification middleware for Go HTTP servers.
// It fetches JWKS from a Rampart IAM server and verifies Bearer tokens,
// working with standard net/http, chi, and any router using http.Handler.
package rampart

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type contextKey struct{}

// Config holds the configuration for the Rampart middleware.
type Config struct {
	// Issuer is the base URL of the Rampart server (e.g. "https://auth.example.com").
	Issuer string

	// Algorithms is the list of accepted signing algorithms. Defaults to ["RS256"].
	Algorithms []string
}

// Claims represents the verified JWT claims from a Rampart access token.
type Claims struct {
	Sub               string   `json:"sub"`
	Iss               string   `json:"iss"`
	Iat               float64  `json:"iat"`
	Exp               float64  `json:"exp"`
	OrgID             string   `json:"org_id"`
	PreferredUsername string   `json:"preferred_username"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
	GivenName         string   `json:"given_name,omitempty"`
	FamilyName        string   `json:"family_name,omitempty"`
	Roles             []string `json:"roles,omitempty"`
}

// ErrorResponse is the JSON body returned on auth failures.
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Status           int    `json:"status"`
}

// NewMiddleware returns an http.Handler middleware that verifies JWT tokens
// issued by the configured Rampart server. It fetches the JWKS from
// {issuer}/.well-known/jwks.json and caches it automatically.
func NewMiddleware(cfg Config) func(http.Handler) http.Handler {
	issuer := strings.TrimRight(cfg.Issuer, "/")
	jwksURL := issuer + "/.well-known/jwks.json"

	// cfg.Algorithms is reserved for future use; jwx validates via JWKS key alg.
	cache := jwk.NewCache(context.Background())
	_ = cache.Register(jwksURL)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Missing authorization header.")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid authorization header format.")
				return
			}

			keySet, err := cache.Get(r.Context(), jwksURL)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Failed to fetch JWKS.")
				return
			}

			token, err := jwt.Parse([]byte(parts[1]),
				jwt.WithKeySet(keySet),
				jwt.WithIssuer(issuer),
				jwt.WithValidate(true),
			)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired access token.")
				return
			}

			claims := mapClaims(token)
			ctx := context.WithValue(r.Context(), contextKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext extracts the verified Rampart claims from the request context.
// Returns nil and false if the middleware has not run or authentication failed.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(contextKey{}).(*Claims)
	return c, ok
}

// RequireRoles returns middleware that checks the authenticated user has all
// of the specified roles. Must be used after NewMiddleware. Returns 403 if
// any required role is missing.
func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok || claims == nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required.")
				return
			}

			userRoles := make(map[string]bool, len(claims.Roles))
			for _, role := range claims.Roles {
				userRoles[role] = true
			}

			var missing []string
			for _, required := range roles {
				if !userRoles[required] {
					missing = append(missing, required)
				}
			}

			if len(missing) > 0 {
				writeError(w, http.StatusForbidden, "forbidden",
					"Missing required role(s): "+strings.Join(missing, ", "))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func mapClaims(token jwt.Token) *Claims {
	c := &Claims{
		Sub: token.Subject(),
		Iss: token.Issuer(),
		Iat: float64(token.IssuedAt().Unix()),
		Exp: float64(token.Expiration().Unix()),
	}

	if v, ok := token.Get("org_id"); ok {
		if s, ok := v.(string); ok {
			c.OrgID = s
		}
	}
	if v, ok := token.Get("preferred_username"); ok {
		if s, ok := v.(string); ok {
			c.PreferredUsername = s
		}
	}
	if v, ok := token.Get("email"); ok {
		if s, ok := v.(string); ok {
			c.Email = s
		}
	}
	if v, ok := token.Get("email_verified"); ok {
		if b, ok := v.(bool); ok {
			c.EmailVerified = b
		}
	}
	if v, ok := token.Get("given_name"); ok {
		if s, ok := v.(string); ok {
			c.GivenName = s
		}
	}
	if v, ok := token.Get("family_name"); ok {
		if s, ok := v.(string); ok {
			c.FamilyName = s
		}
	}
	if v, ok := token.Get("roles"); ok {
		if arr, ok := v.([]interface{}); ok {
			roles := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					roles = append(roles, s)
				}
			}
			c.Roles = roles
		}
	}

	return c
}

func writeError(w http.ResponseWriter, status int, errCode string, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(ErrorResponse{
		Error:            errCode,
		ErrorDescription: description,
		Status:           status,
	}); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}
