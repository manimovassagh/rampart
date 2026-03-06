package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/token"
)

type authUserKey string

const authenticatedUserKey authUserKey = "authenticated_user"

// AuthenticatedUser holds the identity extracted from a verified JWT.
type AuthenticatedUser struct {
	UserID            uuid.UUID
	OrgID             uuid.UUID
	PreferredUsername string
	Email             string
	EmailVerified     bool
	GivenName         string
	FamilyName        string
	Roles             []string
}

// HasRole returns true if the user has the given role.
func (u *AuthenticatedUser) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// Auth returns middleware that verifies RS256 Bearer tokens and stores the user in context.
func Auth(pubKey *rsa.PublicKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeAuthError(w, "Missing authorization header.")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeAuthError(w, "Invalid authorization header format.")
				return
			}

			claims, err := token.VerifyAccessToken(pubKey, parts[1])
			if err != nil {
				writeAuthError(w, "Invalid or expired access token.")
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				writeAuthError(w, "Invalid token subject.")
				return
			}

			authUser := &AuthenticatedUser{
				UserID:            userID,
				OrgID:             claims.OrgID,
				PreferredUsername: claims.PreferredUsername,
				Email:             claims.Email,
				EmailVerified:     claims.EmailVerified,
				GivenName:         claims.GivenName,
				FamilyName:        claims.FamilyName,
				Roles:             claims.Roles,
			}

			ctx := context.WithValue(r.Context(), authenticatedUserKey, authUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that checks the authenticated user has the specified role.
// Must be used after Auth middleware. Returns 403 Forbidden if the role is missing.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetAuthenticatedUser(r.Context())
			if user == nil {
				writeAuthError(w, "Missing authorization header.")
				return
			}
			if !user.HasRole(role) {
				writeForbiddenError(w, "Insufficient permissions. Required role: "+role+".")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SetAuthenticatedUser stores the authenticated user in a context (for testing).
func SetAuthenticatedUser(ctx context.Context, user *AuthenticatedUser) context.Context {
	return context.WithValue(ctx, authenticatedUserKey, user)
}

// GetAuthenticatedUser retrieves the authenticated user from the request context.
func GetAuthenticatedUser(ctx context.Context) *AuthenticatedUser {
	if ctx == nil {
		return nil
	}
	if u, ok := ctx.Value(authenticatedUserKey).(*AuthenticatedUser); ok {
		return u
	}
	return nil
}

// writeAuthError writes a 401 JSON error response without importing apierror (to avoid import cycles).
func writeAuthError(w http.ResponseWriter, description string) {
	writeErrorResponse(w, "unauthorized", description, http.StatusUnauthorized)
}

// writeForbiddenError writes a 403 JSON error response.
func writeForbiddenError(w http.ResponseWriter, description string) {
	writeErrorResponse(w, "forbidden", description, http.StatusForbidden)
}

// writeErrorResponse writes a JSON error response with the given code and status.
func writeErrorResponse(w http.ResponseWriter, code, description string, status int) {
	reqID := w.Header().Get(HeaderRequestID)
	resp := struct {
		Code        string `json:"error"`
		Description string `json:"error_description"`
		Status      int    `json:"status"`
		RequestID   string `json:"request_id,omitempty"`
	}{
		Code:        code,
		Description: description,
		Status:      status,
		RequestID:   reqID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}
