package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// MetricsAuth returns middleware that validates a Bearer token for the /metrics endpoint.
func MetricsAuth(token string) func(http.Handler) http.Handler {
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

			if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(token)) != 1 {
				writeAuthError(w, "Invalid metrics token.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
