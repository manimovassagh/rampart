package webhook

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateWebhookURL checks that the given URL is safe for use as a webhook
// endpoint. It rejects private/internal IPs, non-HTTP(S) schemes, and known
// cloud metadata endpoints to prevent SSRF attacks.
func ValidateWebhookURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got %q", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL must contain a hostname")
	}

	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") {
		return fmt.Errorf("webhook URLs must not target localhost")
	}

	ips, err := net.LookupHost(host)
	if err != nil {
		if ip := net.ParseIP(host); ip != nil {
			if err := checkIP(ip); err != nil {
				return err
			}
			return nil
		}
		return fmt.Errorf("cannot resolve hostname %q: %w", host, err)
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if err := checkIP(ip); err != nil {
			return err
		}
	}

	return nil
}

func checkIP(ip net.IP) error {
	if ip.IsLoopback() {
		return fmt.Errorf("webhook URLs must not target loopback addresses")
	}
	if ip.IsPrivate() {
		return fmt.Errorf("webhook URLs must not target private network addresses")
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("webhook URLs must not target link-local addresses")
	}
	if ip.IsUnspecified() {
		return fmt.Errorf("webhook URLs must not target unspecified (0.0.0.0) addresses")
	}
	metadata := net.ParseIP("169.254.169.254")
	if ip.Equal(metadata) {
		return fmt.Errorf("webhook URLs must not target cloud metadata endpoints")
	}
	return nil
}
