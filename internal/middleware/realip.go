package middleware

import (
	"net"
	"net/http"
	"strings"
)

// TrustedRealIP returns middleware that sets r.RemoteAddr from
// X-Forwarded-For or X-Real-IP headers ONLY when the direct peer
// is in the trusted proxy list.
//
// When trustedCIDRs is empty, proxy headers are never trusted and
// r.RemoteAddr is left as the raw TCP peer address. This prevents
// attackers from spoofing their IP to bypass rate limiting.
func TrustedRealIP(trustedCIDRs []string) func(http.Handler) http.Handler {
	nets := parseCIDRs(trustedCIDRs)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(nets) == 0 {
				// No trusted proxies configured — never trust proxy headers.
				next.ServeHTTP(w, r)
				return
			}

			peerIP := extractIP(r.RemoteAddr)
			if peerIP == nil || !isTrusted(peerIP, nets) {
				// Direct peer is not a trusted proxy — ignore headers.
				next.ServeHTTP(w, r)
				return
			}

			// Peer is trusted — extract client IP from headers.
			if rip := r.Header.Get("X-Real-IP"); rip != "" {
				r.RemoteAddr = rip
			} else if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				// Use the rightmost non-trusted IP in the chain.
				r.RemoteAddr = rightmostNonTrusted(xff, nets)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseCIDRs converts string CIDR/IP entries into net.IPNet slices.
func parseCIDRs(entries []string) []*net.IPNet {
	var nets []*net.IPNet
	for _, entry := range entries {
		if !strings.Contains(entry, "/") {
			// Bare IP — convert to /32 or /128.
			ip := net.ParseIP(entry)
			if ip == nil {
				continue
			}
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			continue
		}
		_, cidr, err := net.ParseCIDR(entry)
		if err != nil {
			continue
		}
		nets = append(nets, cidr)
	}
	return nets
}

// extractIP parses the IP from a host:port string.
func extractIP(addr string) net.IP {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	return net.ParseIP(host)
}

// isTrusted checks if ip is within any of the trusted networks.
func isTrusted(ip net.IP, nets []*net.IPNet) bool {
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// rightmostNonTrusted walks the X-Forwarded-For chain from right to left
// and returns the first IP that is NOT in the trusted proxy list. This is
// the standard algorithm for extracting client IPs behind multiple proxies.
func rightmostNonTrusted(xff string, nets []*net.IPNet) string {
	parts := strings.Split(xff, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		ip := net.ParseIP(strings.TrimSpace(parts[i]))
		if ip == nil {
			continue
		}
		if !isTrusted(ip, nets) {
			return ip.String()
		}
	}
	// All IPs in the chain are trusted — use the leftmost.
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return ""
}
