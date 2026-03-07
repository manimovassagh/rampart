package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersDefaultHeaders(t *testing.T) {
	handler := SecurityHeaders(SecurityHeadersConfig{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	handler.ServeHTTP(rr, req)

	expected := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"X-Xss-Protection":        "1; mode=block",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
		"Content-Security-Policy": "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'",
		"Permissions-Policy":      "camera=(), microphone=(), geolocation=()",
	}

	for header, want := range expected {
		got := rr.Header().Get(header)
		if got != want {
			t.Errorf("header %s = %q, want %q", header, got, want)
		}
	}

	if hsts := rr.Header().Get("Strict-Transport-Security"); hsts != "" {
		t.Errorf("HSTS header should not be set when disabled, got %q", hsts)
	}
}

func TestSecurityHeadersHSTSEnabled(t *testing.T) {
	handler := SecurityHeaders(SecurityHeadersConfig{HSTSEnabled: true})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	handler.ServeHTTP(rr, req)

	want := "max-age=31536000; includeSubDomains"
	got := rr.Header().Get("Strict-Transport-Security")
	if got != want {
		t.Errorf("HSTS header = %q, want %q", got, want)
	}
}

func TestSecurityHeadersHSTSDisabled(t *testing.T) {
	handler := SecurityHeaders(SecurityHeadersConfig{HSTSEnabled: false})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	handler.ServeHTTP(rr, req)

	if hsts := rr.Header().Get("Strict-Transport-Security"); hsts != "" {
		t.Errorf("HSTS header should not be set when disabled, got %q", hsts)
	}
}

func TestSecurityHeadersCallsNext(t *testing.T) {
	called := false
	handler := SecurityHeaders(SecurityHeadersConfig{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("next handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}
