package middleware

import (
	"net/http"
	"strings"
)

// RequireJSON is middleware that returns 415 Unsupported Media Type if the
// request has a body (Content-Length > 0 or Transfer-Encoding) but the
// Content-Type is not application/json.
func RequireJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength == 0 && r.Header.Get("Transfer-Encoding") == "" {
			next.ServeHTTP(w, r)
			return
		}

		ct := r.Header.Get("Content-Type")
		if ct == "" || !strings.HasPrefix(strings.ToLower(ct), "application/json") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnsupportedMediaType)
			_, _ = w.Write([]byte(`{"error":"unsupported_media_type","message":"Content-Type must be application/json."}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
