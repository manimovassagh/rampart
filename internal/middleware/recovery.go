package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery is middleware that recovers from panics, logs the stack trace,
// and returns a 500 Internal Server Error response.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()
					logger.Error("panic recovered",
						"error", err,
						"stack", string(stack),
						"method", r.Method,
						"path", r.URL.Path,
						"request_id", GetRequestID(r.Context()),
					)

					w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal_error","error_description":"An unexpected error occurred.","status":500}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
