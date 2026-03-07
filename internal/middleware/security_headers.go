package middleware

import "net/http"

// Security header constants to avoid string duplication.
const (
	headerXContentTypeOptions = "X-Content-Type-Options"
	headerXFrameOptions       = "X-Frame-Options"
	headerXXSSProtection      = "X-XSS-Protection"
	headerReferrerPolicy      = "Referrer-Policy"
	headerCSP                 = "Content-Security-Policy"
	headerPermissionsPolicy   = "Permissions-Policy"
	headerHSTS                = "Strict-Transport-Security"

	valueNosniff           = "nosniff"
	valueDeny              = "DENY"
	valueXSSBlock          = "1; mode=block"
	valueReferrerPolicy    = "strict-origin-when-cross-origin"
	valueCSP               = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'"
	valuePermissionsPolicy = "camera=(), microphone=(), geolocation=()"
	valueHSTS              = "max-age=31536000; includeSubDomains"
)

// SecurityHeadersConfig controls the behavior of the SecurityHeaders middleware.
type SecurityHeadersConfig struct {
	// HSTSEnabled controls whether the Strict-Transport-Security header is set.
	// Should only be true when TLS termination happens at this server.
	HSTSEnabled bool
}

// SecurityHeaders is middleware that sets common security headers on all responses.
func SecurityHeaders(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set(headerXContentTypeOptions, valueNosniff)
			h.Set(headerXFrameOptions, valueDeny)
			h.Set(headerXXSSProtection, valueXSSBlock)
			h.Set(headerReferrerPolicy, valueReferrerPolicy)
			h.Set(headerCSP, valueCSP)
			h.Set(headerPermissionsPolicy, valuePermissionsPolicy)

			if cfg.HSTSEnabled {
				h.Set(headerHSTS, valueHSTS)
			}

			next.ServeHTTP(w, r)
		})
	}
}
