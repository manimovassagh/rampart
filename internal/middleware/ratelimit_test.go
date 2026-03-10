package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllowsUnderLimit(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)
	defer rl.Close()

	for i := range 5 {
		if !rl.Allow("192.168.1.1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}
}

func TestRateLimiterBlocksOverLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	defer rl.Close()

	for range 3 {
		rl.Allow("10.0.0.1")
	}

	if rl.Allow("10.0.0.1") {
		t.Error("4th request should be blocked")
	}
}

func TestRateLimiterPerIPIsolation(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	defer rl.Close()

	// Exhaust limit for IP A
	rl.Allow("1.1.1.1")
	rl.Allow("1.1.1.1")

	if rl.Allow("1.1.1.1") {
		t.Error("IP A should be blocked")
	}

	// IP B should still be allowed
	if !rl.Allow("2.2.2.2") {
		t.Error("IP B should not be affected by IP A")
	}
}

func TestRateLimiterSlidingWindow(t *testing.T) {
	rl := NewRateLimiter(2, 50*time.Millisecond)
	defer rl.Close()

	rl.Allow("10.0.0.1")
	rl.Allow("10.0.0.1")

	if rl.Allow("10.0.0.1") {
		t.Error("should be blocked before window expires")
	}

	// Wait for window to pass
	time.Sleep(60 * time.Millisecond)

	if !rl.Allow("10.0.0.1") {
		t.Error("should be allowed after window expires")
	}
}

func TestRateLimiterMiddlewareReturns429(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	defer rl.Close()

	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request — allowed
	req := httptest.NewRequest(http.MethodPost, "/login", http.NoBody)
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("first request status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Second request — blocked
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("second request status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}

	// Verify JSON error body
	var errResp rateLimitErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.Code != "rate_limit_exceeded" {
		t.Errorf("error code = %q, want rate_limit_exceeded", errResp.Code)
	}
	if errResp.Status != http.StatusTooManyRequests {
		t.Errorf("error status = %d, want %d", errResp.Status, http.StatusTooManyRequests)
	}

	// Verify Retry-After header
	if rec.Header().Get("Retry-After") == "" {
		t.Error("missing Retry-After header on 429 response")
	}
}

func TestRateLimiterIgnoresXForwardedFor(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	defer rl.Close()

	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Two requests with different X-Forwarded-For but same RemoteAddr.
	// The rate limiter should use RemoteAddr only (set by chi's RealIP middleware).
	req1 := httptest.NewRequest(http.MethodPost, "/login", http.NoBody)
	req1.Header.Set("X-Forwarded-For", "203.0.113.1")
	req1.RemoteAddr = "10.0.0.1:9999"

	req2 := httptest.NewRequest(http.MethodPost, "/login", http.NoBody)
	req2.Header.Set("X-Forwarded-For", "203.0.113.2")
	req2.RemoteAddr = "10.0.0.1:9999"

	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Errorf("req1 status = %d, want 200", rec1.Code)
	}

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("req2 should be rate-limited (same RemoteAddr), got %d", rec2.Code)
	}
}

func TestRateLimiterStaleCleanup(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	defer rl.Close()

	rl.Allow("stale-ip")

	// Manually age the entry
	rl.mu.Lock()
	rl.entries["stale-ip"].lastSeen = time.Now().Add(-rateLimitEntryTTL - time.Second)
	rl.mu.Unlock()

	rl.removeStale()

	rl.mu.Lock()
	_, exists := rl.entries["stale-ip"]
	rl.mu.Unlock()

	if exists {
		t.Error("stale entry should have been cleaned up")
	}
}

func TestClientIPUsesOnlyRemoteAddr(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		wantIP     string
	}{
		{
			name:       "RemoteAddrWithPort",
			remoteAddr: "192.168.1.1:12345",
			wantIP:     "192.168.1.1",
		},
		{
			name:       "IgnoresXForwardedFor",
			remoteAddr: "10.0.0.1:9999",
			xff:        "203.0.113.50",
			wantIP:     "10.0.0.1",
		},
		{
			name:       "IgnoresXRealIP",
			remoteAddr: "10.0.0.1:9999",
			xri:        "198.51.100.1",
			wantIP:     "10.0.0.1",
		},
		{
			name:       "RemoteAddrNoPort",
			remoteAddr: "192.168.1.1",
			wantIP:     "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-Ip", tt.xri)
			}

			got := clientIP(req)
			if got != tt.wantIP {
				t.Errorf("clientIP() = %q, want %q", got, tt.wantIP)
			}
		})
	}
}
