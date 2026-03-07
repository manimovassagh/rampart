package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	cfg := &Config{
		Issuer:      "http://localhost:8080",
		AccessToken: "my-token",
	}

	c := NewClient(cfg)

	if c.BaseURL != cfg.Issuer {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, cfg.Issuer)
	}
	if c.Token != cfg.AccessToken {
		t.Errorf("Token = %q, want %q", c.Token, cfg.AccessToken)
	}
	if c.HTTPClient == nil {
		t.Fatal("HTTPClient is nil")
	}
}

func TestClientGetSuccess(t *testing.T) {
	expected := map[string]string{"hello": "world"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/test" {
			t.Errorf("path = %s, want /api/test", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(expected); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	var result map[string]string
	if err := c.Get("/api/test", &result); err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if result["hello"] != "world" {
		t.Errorf("result = %v, want hello:world", result)
	}
}

func TestClientPostSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["name"] != "test" {
			t.Errorf("body name = %q, want test", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"id": "123"}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	var result map[string]string
	reqBody := map[string]string{"name": "test"}
	if err := c.Post("/api/create", reqBody, &result); err != nil {
		t.Fatalf("Post error: %v", err)
	}
	if result["id"] != "123" {
		t.Errorf("result id = %q, want 123", result["id"])
	}
}

func TestClientDeleteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"deleted":true}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	var result map[string]bool
	if err := c.Delete("/api/item/1", &result); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if !result["deleted"] {
		t.Error("expected deleted=true")
	}
}

func TestClientTokenHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-secret-token" {
			t.Errorf("Authorization = %q, want Bearer my-secret-token", auth)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, Token: "my-secret-token", HTTPClient: srv.Client()}
	if err := c.Get("/api/protected", nil); err != nil {
		t.Fatalf("Get error: %v", err)
	}
}

func TestClientNoTokenHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, Token: "", HTTPClient: srv.Client()}
	if err := c.Get("/api/public", nil); err != nil {
		t.Fatalf("Get error: %v", err)
	}
}

func TestClientDoHTTP400WithErrorDescription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_request",
			"error_description": "username is required",
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	err := c.Get("/api/fail", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "username is required (HTTP 400)" {
		t.Errorf("error = %q, want 'username is required (HTTP 400)'", got)
	}
}

func TestClientDoHTTP401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	err := c.Get("/api/secret", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClientDoHTTP500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	err := c.Get("/api/broken", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "HTTP 500: internal server error" {
		t.Errorf("error = %q, want 'HTTP 500: internal server error'", got)
	}
}

func TestClientDoHTTP400PlainTextError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	err := c.Get("/api/fail", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "HTTP 400: bad request" {
		t.Errorf("error = %q, want 'HTTP 400: bad request'", got)
	}
}

func TestClientDoMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	var result map[string]string
	err := c.Get("/api/bad-json", &result)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestClientDoNilResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":"ignored"}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	if err := c.Get("/api/no-decode", nil); err != nil {
		t.Fatalf("expected no error when result is nil, got: %v", err)
	}
}

func TestClientPostNilBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "" {
			t.Errorf("Content-Type should be empty for nil body, got %q", ct)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	var result map[string]bool
	if err := c.Post("/api/action", nil, &result); err != nil {
		t.Fatalf("Post error: %v", err)
	}
	if !result["ok"] {
		t.Error("expected ok=true")
	}
}

func TestClientDoInvalidURL(t *testing.T) {
	c := &Client{BaseURL: "http://[invalid", HTTPClient: http.DefaultClient}
	err := c.Get("/path", nil)
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestClientDoServerDown(t *testing.T) {
	c := &Client{BaseURL: "http://127.0.0.1:1", HTTPClient: http.DefaultClient}
	err := c.Get("/api/test", nil)
	if err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}
}
