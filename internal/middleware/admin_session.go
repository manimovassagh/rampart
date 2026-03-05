package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/token"
)

const (
	sessionCookieName = "rampart_admin_session"
	csrfCookieName    = "rampart_csrf"
	csrfFieldName     = "csrf_token"
	flashCookieName   = "rampart_flash"
)

// AdminSession returns middleware that validates admin session cookies.
// Unauthenticated requests are redirected to the admin login page.
func AdminSession(pubKey *rsa.PublicKey, hmacKey []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil || cookie.Value == "" {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
				return
			}

			accessToken, ok := verifySignedCookie(cookie.Value, hmacKey)
			if !ok {
				ClearAdminSession(w)
				http.Redirect(w, r, "/admin/login", http.StatusFound)
				return
			}

			claims, err := token.VerifyAccessToken(pubKey, accessToken)
			if err != nil {
				ClearAdminSession(w)
				http.Redirect(w, r, "/admin/login", http.StatusFound)
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				ClearAdminSession(w)
				http.Redirect(w, r, "/admin/login", http.StatusFound)
				return
			}

			authUser := &AuthenticatedUser{
				UserID:           userID,
				OrgID:            claims.OrgID,
				PreferredUsername: claims.PreferredUsername,
				Email:            claims.Email,
				EmailVerified:    claims.EmailVerified,
				GivenName:        claims.GivenName,
				FamilyName:       claims.FamilyName,
			}

			ctx := context.WithValue(r.Context(), authenticatedUserKey, authUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CSRFProtect returns middleware that validates CSRF tokens on state-changing requests.
func CSRFProtect() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				ensureCSRFCookie(w, r)
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(csrfCookieName)
			if err != nil || cookie.Value == "" {
				http.Error(w, "CSRF validation failed.", http.StatusForbidden)
				return
			}

			var formToken string
			if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" ||
				strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				if err := r.ParseForm(); err != nil {
					http.Error(w, "Invalid form data.", http.StatusBadRequest)
					return
				}
				formToken = r.FormValue(csrfFieldName)
			}
			if formToken == "" {
				formToken = r.Header.Get("X-CSRF-Token")
			}

			if !hmac.Equal([]byte(cookie.Value), []byte(formToken)) {
				http.Error(w, "CSRF validation failed.", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SetAdminSession creates a signed httpOnly cookie with the access token.
func SetAdminSession(w http.ResponseWriter, accessToken string, hmacKey []byte, maxAge int) {
	signed := signCookie(accessToken, hmacKey)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    signed,
		Path:     "/admin/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // Set true in production via config
	})
}

// ClearAdminSession removes the admin session cookie.
func ClearAdminSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/admin/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// SetFlash sets a flash message cookie that will be read and cleared on the next request.
func SetFlash(w http.ResponseWriter, message string) {
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    base64.URLEncoding.EncodeToString([]byte(message)),
		Path:     "/admin/",
		MaxAge:   10,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// GetFlash reads and clears the flash message from the cookie.
func GetFlash(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie(flashCookieName)
	if err != nil || cookie.Value == "" {
		return ""
	}

	// Clear the cookie immediately
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    "",
		Path:     "/admin/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	decoded, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return ""
	}
	return string(decoded)
}

// GetCSRFToken returns the current CSRF token from the cookie.
func GetCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// GenerateHMACKey creates a random 32-byte HMAC key.
func GenerateHMACKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generating HMAC key: %w", err)
	}
	return key, nil
}

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(csrfCookieName); err == nil {
		return
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return
	}
	tok := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    tok,
		Path:     "/admin/",
		MaxAge:   3600,
		HttpOnly: false, // Must be readable by forms via template
		SameSite: http.SameSiteLaxMode,
	})
}

// signCookie signs the value with HMAC-SHA256 and returns "signature.value".
func signCookie(value string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(value))
	sig := hex.EncodeToString(mac.Sum(nil))
	return sig + "." + value
}

// verifySignedCookie verifies and extracts the value from a "signature.value" cookie.
func verifySignedCookie(signed string, key []byte) (string, bool) {
	idx := strings.IndexByte(signed, '.')
	if idx < 1 {
		return "", false
	}
	sig := signed[:idx]
	value := signed[idx+1:]

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(value))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", false
	}
	return value, true
}
