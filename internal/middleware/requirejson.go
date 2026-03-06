package middleware

import (
	"net/http"
	"strings"
)

// RequireJSON returns middleware that enforces Content-Type: application/json
// on requests with bodies (POST, PUT, PATCH). Requests without bodies or
// with multipart/form-data are passed through unchanged.
func RequireJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			ct := r.Header.Get("Content-Type")
			mediaType := strings.TrimSpace(strings.SplitN(ct, ";", 2)[0])
			if !strings.EqualFold(mediaType, "application/json") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				_, _ = w.Write([]byte(`{"error":"unsupported_media_type","error_description":"Content-Type must be application/json","status":415}` + "\n"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
