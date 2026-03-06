package builtin

import (
	"context"
	"sync"
	"time"

	"github.com/manimovassagh/rampart/internal/plugin"
)

const (
	defaultMaxAttempts = 10
	defaultWindow      = 5 * time.Minute
	rateLimitName      = "builtin-rate-limiter"
	rateLimitVersion   = "1.0.0"
	rateLimitDesc      = "Blocks IPs with too many failed login attempts"
	rateLimitReason    = "too many failed login attempts, try again later"
)

// attempt tracks failed login attempts from an IP.
type attempt struct {
	count     int
	firstSeen time.Time
}

// RateLimiter is a built-in plugin that rate-limits authentication
// attempts by IP address using a sliding window.
type RateLimiter struct {
	maxAttempts int
	window      time.Duration
	attempts    sync.Map // map[string]*attempt
}

// NewRateLimiter creates a new rate limiter plugin with default settings.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		maxAttempts: defaultMaxAttempts,
		window:      defaultWindow,
	}
}

func (rl *RateLimiter) Name() string        { return rateLimitName }
func (rl *RateLimiter) Version() string      { return rateLimitVersion }
func (rl *RateLimiter) Description() string  { return rateLimitDesc }

func (rl *RateLimiter) Init(_ context.Context, config map[string]any) error {
	if config == nil {
		return nil
	}
	if v, ok := config["max_attempts"]; ok {
		if n, ok := v.(int); ok && n > 0 {
			rl.maxAttempts = n
		}
	}
	if v, ok := config["window_seconds"]; ok {
		if n, ok := v.(int); ok && n > 0 {
			rl.window = time.Duration(n) * time.Second
		}
	}
	return nil
}

func (rl *RateLimiter) Shutdown(_ context.Context) error {
	return nil
}

func (rl *RateLimiter) Hooks() []plugin.HookPoint {
	return []plugin.HookPoint{plugin.HookPreAuth, plugin.HookPostAuth}
}

// OnPreAuth checks if the IP has exceeded the rate limit.
func (rl *RateLimiter) OnPreAuth(_ context.Context, req *plugin.AuthRequest) (*plugin.AuthResponse, error) {
	if req == nil || req.IP == "" {
		return &plugin.AuthResponse{Allow: true}, nil
	}

	raw, ok := rl.attempts.Load(req.IP)
	if !ok {
		return &plugin.AuthResponse{Allow: true}, nil
	}

	a := raw.(*attempt)
	if time.Since(a.firstSeen) > rl.window {
		rl.attempts.Delete(req.IP)
		return &plugin.AuthResponse{Allow: true}, nil
	}

	if a.count >= rl.maxAttempts {
		return &plugin.AuthResponse{
			Allow:  false,
			Reason: rateLimitReason,
		}, nil
	}

	return &plugin.AuthResponse{Allow: true}, nil
}

// OnPostAuth tracks failed login attempts per IP.
func (rl *RateLimiter) OnPostAuth(_ context.Context, req *plugin.AuthRequest, result *plugin.AuthResult) error {
	if req == nil || req.IP == "" || result == nil {
		return nil
	}

	// On success, clear the counter for this IP
	if result.Success {
		rl.attempts.Delete(req.IP)
		return nil
	}

	// On failure, increment the counter
	now := time.Now()
	raw, loaded := rl.attempts.LoadOrStore(req.IP, &attempt{count: 1, firstSeen: now})
	if loaded {
		a := raw.(*attempt)
		if time.Since(a.firstSeen) > rl.window {
			// Window expired, reset
			rl.attempts.Store(req.IP, &attempt{count: 1, firstSeen: now})
		} else {
			a.count++
		}
	}

	return nil
}
