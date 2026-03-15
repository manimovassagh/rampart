// Package rampart provides JWT verification middleware for Go HTTP servers.
//
// It validates RS256 tokens issued by a Rampart IAM server, extracts typed
// claims, and supports role-based access control. The middleware fetches
// JWKS from the issuer's /.well-known/jwks.json endpoint and caches the
// key set automatically.
//
// It works with standard [net/http], chi, gorilla/mux, and any router
// that accepts the [http.Handler] interface.
//
// # Quick Start
//
//	auth := rampart.NewMiddleware(rampart.Config{
//		Issuer: "https://auth.example.com",
//	})
//	mux.Handle("/api/profile", auth(handler))
//
// # Claims
//
// After successful verification, claims are available via [ClaimsFromContext]:
//
//	claims, ok := rampart.ClaimsFromContext(r.Context())
//	fmt.Println(claims.Email, claims.Roles)
//
// # Role-Based Access Control
//
// Chain [RequireRoles] after the auth middleware to enforce role checks:
//
//	adminOnly := rampart.RequireRoles("admin")
//	mux.Handle("/admin", auth(adminOnly(handler)))
//
// # Error Handling
//
// Both [NewMiddleware] and [RequireRoles] return JSON error responses using
// the [ErrorResponse] struct. The auth middleware returns 401 Unauthorized;
// the role middleware returns 403 Forbidden.
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

// Config holds the configuration for the Rampart authentication middleware.
//
// At minimum, [Config.Issuer] must be set. The middleware uses it to locate
// the JWKS endpoint at {Issuer}/.well-known/jwks.json.
type Config struct {
	// Issuer is the base URL of the Rampart server, without a trailing slash
	// (e.g., "https://auth.example.com"). The middleware appends
	// /.well-known/jwks.json to discover signing keys and validates the
	// "iss" claim in every token against this value.
	Issuer string

	// Algorithms is the list of accepted signing algorithms.
	// Reserved for future use; the current implementation relies on the
	// algorithm declared in the JWKS key set.
	Algorithms []string
}

// Claims represents the verified JWT claims extracted from a Rampart access
// token. Obtain it from a request context with [ClaimsFromContext].
//
// Fields map to the JSON claim names shown in their struct tags.
// Optional fields (GivenName, FamilyName, Roles) use the omitempty tag
// and will be zero-valued when absent from the token.
type Claims struct {
	// Sub is the subject identifier (user UUID) — JWT "sub" claim.
	Sub string `json:"sub"`

	// Iss is the issuer URL — JWT "iss" claim.
	Iss string `json:"iss"`

	// Iat is the token issued-at time as a Unix timestamp — JWT "iat" claim.
	Iat float64 `json:"iat"`

	// Exp is the token expiration time as a Unix timestamp — JWT "exp" claim.
	Exp float64 `json:"exp"`

	// OrgID is the organization UUID the user belongs to — custom "org_id" claim.
	OrgID string `json:"org_id"`

	// PreferredUsername is the user's display name — OIDC "preferred_username" claim.
	PreferredUsername string `json:"preferred_username"`

	// Email is the user's email address — OIDC "email" claim.
	Email string `json:"email"`

	// EmailVerified indicates whether the email address has been verified —
	// OIDC "email_verified" claim.
	EmailVerified bool `json:"email_verified"`

	// GivenName is the user's first name, if provided — OIDC "given_name" claim.
	GivenName string `json:"given_name,omitempty"`

	// FamilyName is the user's last name, if provided — OIDC "family_name" claim.
	FamilyName string `json:"family_name,omitempty"`

	// Roles is the set of roles assigned to the user — custom "roles" claim.
	// Use [RequireRoles] for declarative role checks, or inspect this field
	// directly for finer-grained authorization logic.
	Roles []string `json:"roles,omitempty"`
}

// ErrorResponse is the JSON body returned by the middleware on authentication
// or authorization failures. It follows the Rampart server error format.
type ErrorResponse struct {
	// Error is a machine-readable error code (e.g., "unauthorized", "forbidden").
	Error string `json:"error"`

	// ErrorDescription is a human-readable explanation of the failure.
	ErrorDescription string `json:"error_description"`

	// Status is the HTTP status code (e.g., 401, 403).
	Status int `json:"status"`
}

// NewMiddleware returns an [http.Handler] middleware that verifies JWT Bearer
// tokens issued by the configured Rampart server.
//
// On each request the middleware:
//  1. Extracts the Bearer token from the Authorization header.
//  2. Fetches (and caches) the JWKS from {Issuer}/.well-known/jwks.json.
//  3. Validates the token signature, issuer, and expiration.
//  4. Stores the parsed [Claims] in the request context for downstream handlers.
//
// If any step fails, the middleware writes a 401 JSON [ErrorResponse] and
// does not call the next handler.
//
// Example:
//
//	auth := rampart.NewMiddleware(rampart.Config{
//		Issuer: "https://auth.example.com",
//	})
//
//	mux := http.NewServeMux()
//	mux.Handle("/api/profile", auth(http.HandlerFunc(profileHandler)))
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

// ClaimsFromContext extracts the verified [Claims] from the request context.
// It returns the claims and true on success, or nil and false if
// [NewMiddleware] has not run or authentication failed.
//
// Example:
//
//	claims, ok := rampart.ClaimsFromContext(r.Context())
//	if !ok {
//		http.Error(w, "not authenticated", http.StatusUnauthorized)
//		return
//	}
//	fmt.Fprintf(w, "Hello, %s", claims.PreferredUsername)
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(contextKey{}).(*Claims)
	return c, ok
}

// RequireRoles returns middleware that enforces role-based access control.
// It checks that the authenticated user possesses every role listed in roles.
//
// RequireRoles must be chained after [NewMiddleware]; if no [Claims] are
// found in the context it returns 401 Unauthorized. If the user is
// authenticated but lacks one or more required roles it returns 403 Forbidden
// with the missing role names in the error description.
//
// Example:
//
//	auth := rampart.NewMiddleware(rampart.Config{Issuer: issuer})
//	adminOnly := rampart.RequireRoles("admin")
//	editorOrAdmin := rampart.RequireRoles("editor", "admin")
//
//	mux.Handle("/admin", auth(adminOnly(handler)))
//	mux.Handle("/publish", auth(editorOrAdmin(handler)))
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
		if arr, ok := v.([]any); ok {
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
