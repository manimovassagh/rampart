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

	// minResponseDuration prevents timing side-channels on rate-limited
	// responses.  Handlers already pad their own responses to this floor;
	// the rate limiter must do the same so that a 429 is indistinguishable
	// (by timing) from a normal auth error.
	minResponseDuration = 250 * time.Millisecond
)

// rateLimitEntry tracks request timestamps for a single IP.
type rateLimitEntry struct {
	timestamps []time.Time
	lastSeen   time.Time
}

// RateLimiter provides per-IP rate limiting using a sliding window.
//
// IMPORTANT: This is an in-memory, single-instance rate limiter. Each server
// instance maintains its own independent counters, so when running K instances
// behind a load balancer the effective rate limit becomes K × the configured
// limit (requests may be distributed across instances).
//
// For multi-instance / high-availability deployments, enforce rate limiting at
// the ingress layer instead (e.g., nginx limit_req, AWS WAF, Cloudflare Rate
// Limiting, or your cloud load balancer). This keeps the application stateless
// and avoids the complexity of a distributed rate-limiting backend.
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
				start := time.Now()
				writeRateLimitError(w)
				// Pad response time so a 429 is indistinguishable (by timing)
				// from a normal authentication error, preventing user enumeration
				// via timing side-channel.
				if elapsed := time.Since(start); elapsed < minResponseDuration {
					time.Sleep(minResponseDuration - elapsed)
				}
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the client IP from r.RemoteAddr, which chi's RealIP
// middleware has already set from trusted proxy headers.
// We do NOT read X-Forwarded-For or X-Real-Ip directly because an attacker
// can spoof those headers to bypass rate limiting.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
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
