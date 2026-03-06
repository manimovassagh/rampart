package middleware

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	rateLimitCleanupInterval = 5 * time.Minute
	rateLimitEntryTTL        = 2 * time.Minute
)

// rateLimitEntry tracks request timestamps for a single IP.
type rateLimitEntry struct {
	timestamps []time.Time
	lastSeen   time.Time
}

// RateLimiter provides per-IP rate limiting using a sliding window.
type RateLimiter struct {
	mu       sync.Mutex
	entries  map[string]*rateLimitEntry
	limit    int
	window   time.Duration
	stopOnce sync.Once
	stop     chan struct{}
}

// NewRateLimiter creates a rate limiter that allows `limit` requests per `window` per IP.
// It starts a background goroutine to clean up stale entries.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		entries: make(map[string]*rateLimitEntry),
		limit:   limit,
		window:  window,
		stop:    make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Close stops the background cleanup goroutine.
func (rl *RateLimiter) Close() {
	rl.stopOnce.Do(func() {
		close(rl.stop)
	})
}

// Allow checks whether a request from the given IP should be allowed.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	entry, exists := rl.entries[ip]
	if !exists {
		rl.entries[ip] = &rateLimitEntry{
			timestamps: []time.Time{now},
			lastSeen:   now,
		}
		return true
	}

	// Remove timestamps outside the window
	valid := entry.timestamps[:0]
	for _, ts := range entry.timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}
	entry.timestamps = valid
	entry.lastSeen = now

	if len(entry.timestamps) >= rl.limit {
		return false
	}

	entry.timestamps = append(entry.timestamps, now)
	return true
}

// cleanup periodically removes stale entries to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rateLimitCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			rl.removeStale()
		}
	}
}

func (rl *RateLimiter) removeStale() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rateLimitEntryTTL)
	for ip, entry := range rl.entries {
		if entry.lastSeen.Before(cutoff) {
			delete(rl.entries, ip)
		}
	}
}

// Middleware returns an HTTP middleware that enforces the rate limit.
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			if !rl.Allow(ip) {
				writeRateLimitError(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the client IP address from the request.
// It checks X-Forwarded-For first (for proxied environments), then falls
// back to X-Real-Ip, and finally RemoteAddr.
func clientIP(r *http.Request) string {
	// chi's RealIP middleware sets RemoteAddr from X-Forwarded-For/X-Real-Ip,
	// but we also handle it here for defense in depth.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first (leftmost) IP — the original client
		if i := indexOf(xff, ','); i >= 0 {
			xff = xff[:i]
		}
		if ip := trimSpace(xff); ip != "" {
			return ip
		}
	}

	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return trimSpace(xri)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// indexOf returns the index of the first occurrence of sep in s, or -1.
func indexOf(s string, sep byte) int {
	for i := range len(s) {
		if s[i] == sep {
			return i
		}
	}
	return -1
}

// rateLimitErrorResponse matches the apierror.Error structure to avoid import cycles.
type rateLimitErrorResponse struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
	Status      int    `json:"status"`
	RequestID   string `json:"request_id,omitempty"`
}

func writeRateLimitError(w http.ResponseWriter) {
	reqID := w.Header().Get(HeaderRequestID)
	resp := &rateLimitErrorResponse{
		Code:        "rate_limit_exceeded",
		Description: "Rate limit exceeded. Try again later.",
		Status:      http.StatusTooManyRequests,
		RequestID:   reqID,
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "60")
	w.WriteHeader(http.StatusTooManyRequests)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode rate limit error response", "error", err)
	}
}

// trimSpace trims leading and trailing ASCII whitespace.
func trimSpace(s string) string {
	start := 0
	for start < len(s) && s[start] == ' ' {
		start++
	}
	end := len(s)
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}
