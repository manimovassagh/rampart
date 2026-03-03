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
			}

			ctx := context.WithValue(r.Context(), authenticatedUserKey, authUser)
			next.ServeHTTP(w, r.WithContext(ctx))
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
	reqID := w.Header().Get(HeaderRequestID)
	resp := struct {
		Code        string `json:"error"`
		Description string `json:"error_description"`
		Status      int    `json:"status"`
		RequestID   string `json:"request_id,omitempty"`
	}{
		Code:        "unauthorized",
		Description: description,
		Status:      http.StatusUnauthorized,
		RequestID:   reqID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode auth error response", "error", err)
	}
}
