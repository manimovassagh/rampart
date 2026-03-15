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
	"sync/atomic"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/token"
)

const (
	sessionCookieName = "rampart_admin_session"
	csrfCookieName    = "rampart_csrf"
	csrfFieldName     = "csrf_token"
	flashCookieName   = "rampart_flash"

	// AdminLoginPath is the redirect target for unauthenticated admin requests.
	AdminLoginPath = "/admin/login"
	// AdminCookiePath is the cookie path for admin session cookies.
	AdminCookiePath = "/admin/"
)

// secureCookies controls whether cookies are sent with the Secure flag.
// Must be true in production (requires HTTPS). Defaults to false for development.
var secureCookies atomic.Bool

// SetSecureCookies configures whether all cookies should have the Secure flag set.
// Call this at startup before serving requests.
func SetSecureCookies(secure bool) {
	secureCookies.Store(secure)
}

// SecureCookiesEnabled returns whether cookies should have the Secure flag set.
func SecureCookiesEnabled() bool {
	return secureCookies.Load()
}

// AdminSession returns middleware that validates admin session cookies.
// Unauthenticated requests are redirected to the admin login page.
func AdminSession(pubKey *rsa.PublicKey, hmacKey []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil || cookie.Value == "" {
				http.Redirect(w, r, AdminLoginPath, http.StatusFound)
				return
			}

			accessToken, ok := verifySignedCookie(cookie.Value, hmacKey)
			if !ok {
				ClearAdminSession(w)
				http.Redirect(w, r, AdminLoginPath, http.StatusFound)
				return
			}

			claims, err := token.VerifyAccessToken(pubKey, accessToken)
			if err != nil {
				ClearAdminSession(w)
				http.Redirect(w, r, AdminLoginPath, http.StatusFound)
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				ClearAdminSession(w)
				http.Redirect(w, r, AdminLoginPath, http.StatusFound)
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

// RequireAdminSession returns middleware that checks the authenticated user has the "admin" role.
// Must be used after AdminSession middleware. Redirects to login if the role is missing.
func RequireAdminSession() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetAuthenticatedUser(r.Context())
			if user == nil {
				http.Redirect(w, r, AdminLoginPath, http.StatusFound)
				return
			}
			if !user.HasRole("admin") {
				ClearAdminSession(w)
				http.Error(w, "Forbidden: admin role required.", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
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
		Path:     AdminCookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookies.Load(),
	})
}

// ClearAdminSession removes the admin session cookie.
func ClearAdminSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     AdminCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookies.Load(),
	})
}

// SetFlash sets a flash message cookie that will be read and cleared on the next request.
func SetFlash(w http.ResponseWriter, message string) {
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    base64.URLEncoding.EncodeToString([]byte(message)),
		Path:     AdminCookiePath,
		MaxAge:   10,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookies.Load(),
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
		Path:     AdminCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookies.Load(),
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

// OAuthCSRFCookieName is the cookie name for CSRF on the OAuth login form.
const OAuthCSRFCookieName = "rampart_oauth_csrf"

// GenerateCSRFToken creates a new random CSRF token string.
func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating CSRF token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// SetOAuthCSRFCookie sets a CSRF cookie scoped to /oauth/ for the login form.
func SetOAuthCSRFCookie(w http.ResponseWriter, csrfToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthCSRFCookieName,
		Value:    csrfToken,
		Path:     "/oauth/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secureCookies.Load(),
	})
}

const oauthConsentUserCookie = "rampart_consent_uid"

// SetConsentUserCookie stores the authenticated user's ID in an HMAC-signed
// HttpOnly cookie during the consent flow. This prevents user_id forgery.
func SetConsentUserCookie(w http.ResponseWriter, userID uuid.UUID, hmacKey []byte) {
	signed := signCookie(userID.String(), hmacKey)
	http.SetCookie(w, &http.Cookie{
		Name:     oauthConsentUserCookie,
		Value:    signed,
		Path:     "/oauth/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secureCookies.Load(),
	})
}

// GetConsentUserID reads and verifies the HMAC-signed consent cookie.
// Returns uuid.Nil if the cookie is missing, the signature is invalid, or the
// value is not a valid UUID.
func GetConsentUserID(r *http.Request, hmacKey []byte) uuid.UUID {
	cookie, err := r.Cookie(oauthConsentUserCookie)
	if err != nil || cookie.Value == "" {
		return uuid.Nil
	}
	value, ok := verifySignedCookie(cookie.Value, hmacKey)
	if !ok {
		return uuid.Nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// ClearConsentUserCookie removes the consent user cookie.
func ClearConsentUserCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthConsentUserCookie,
		Value:    "",
		Path:     "/oauth/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secureCookies.Load(),
	})
}

// ValidateOAuthCSRF checks the CSRF token from the form against the cookie.
// Returns true if the tokens match.
func ValidateOAuthCSRF(r *http.Request, formToken string) bool {
	cookie, err := r.Cookie(OAuthCSRFCookieName)
	if err != nil || cookie.Value == "" || formToken == "" {
		return false
	}
	return hmac.Equal([]byte(cookie.Value), []byte(formToken))
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
		Path:     AdminCookiePath,
		MaxAge:   3600,
		HttpOnly: true, // Token is injected server-side via Go templates
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookies.Load(),
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
