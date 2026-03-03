package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDGeneratesNew(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id == "" {
			t.Error("expected non-empty request ID in context")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	got := w.Header().Get(HeaderRequestID)
	if got == "" {
		t.Error("expected X-Request-Id response header")
	}
}

func TestRequestIDPropagatesExisting(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id != "my-custom-id" {
			t.Errorf("context request ID = %q, want my-custom-id", id)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set(HeaderRequestID, "my-custom-id")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get(HeaderRequestID); got != "my-custom-id" {
		t.Errorf("X-Request-Id = %q, want my-custom-id", got)
	}
}

func TestGetRequestIDEmptyContext(t *testing.T) {
	if id := GetRequestID(context.TODO()); id != "" {
		t.Errorf("GetRequestID(empty) = %q, want empty", id)
	}
}

//nolint:staticcheck // SA1012: intentionally testing nil context for safety
func TestGetRequestIDNilContext(t *testing.T) {
	if id := GetRequestID(nil); id != "" {
		t.Errorf("GetRequestID(nil) = %q, want empty", id)
	}
}
