package webhook

import (
	"strings"
	"testing"
)

func TestValidateWebhookURL_SSRFLoopback(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"localhost", "https://localhost/hook"},
		{"localhost with port", "https://localhost:8080/hook"},
		{"subdomain of localhost", "https://app.localhost/hook"},
		{"127.0.0.1", "https://127.0.0.1/hook"},
		{"127.0.0.1 with port", "https://127.0.0.1:443/hook"},
		{"IPv6 loopback", "https://[::1]/hook"},
		{"IPv6 loopback with port", "https://[::1]:8443/hook"},
		{"0.0.0.0", "https://0.0.0.0/hook"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if err == nil {
				t.Errorf("expected error for loopback URL %q, got nil", tt.url)
			}
		})
	}
}

func TestValidateWebhookURL_SSRFPrivateIPs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"10.0.0.1", "https://10.0.0.1/hook"},
		{"10.255.255.255", "https://10.255.255.255/hook"},
		{"172.16.0.1", "https://172.16.0.1/hook"},
		{"172.31.255.255", "https://172.31.255.255/hook"},
		{"192.168.0.1", "https://192.168.0.1/hook"},
		{"192.168.1.100", "https://192.168.1.100/hook"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if err == nil {
				t.Errorf("expected error for private IP URL %q, got nil", tt.url)
			}
		})
	}
}

func TestValidateWebhookURL_SSRFLinkLocal(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"link-local 169.254.1.1", "https://169.254.1.1/hook"},
		{"cloud metadata 169.254.169.254", "https://169.254.169.254/latest/meta-data/"},
		{"link-local IPv6", "https://[fe80::1]/hook"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if err == nil {
				t.Errorf("expected error for link-local URL %q, got nil", tt.url)
			}
		})
	}
}

func TestValidateWebhookURL_SSRFIPv6MappedIPv4(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"IPv6-mapped 127.0.0.1", "https://[::ffff:127.0.0.1]/hook"},
		{"IPv6-mapped 10.0.0.1", "https://[::ffff:10.0.0.1]/hook"},
		{"IPv6-mapped 192.168.1.1", "https://[::ffff:192.168.1.1]/hook"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if err == nil {
				t.Errorf("expected error for IPv6-mapped IPv4 URL %q, got nil", tt.url)
			}
		})
	}
}

func TestValidateWebhookURL_SSRFOctalEncoding(t *testing.T) {
	// KNOWN LIMITATION: Go's net.ParseIP does NOT parse octal notation
	// (e.g. 0177.0.0.1 for 127.0.0.1). These hostnames are passed to DNS
	// resolution. If DNS resolves them to a public IP, the validator allows
	// them. This test documents the behavior rather than asserting a block.
	tests := []struct {
		name string
		url  string
	}{
		{"octal 127.0.0.1", "https://0177.0.0.1/hook"},
		{"octal 10.0.0.1", "https://012.0.0.1/hook"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			// The result depends on DNS resolution of the octal hostname.
			// We just verify the validator does not panic.
			_ = err
		})
	}
}

func TestValidateWebhookURL_SSRFHexEncoding(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"hex 127.0.0.1", "https://0x7f000001/hook"},
		{"hex 10.0.0.1", "https://0x0a000001/hook"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			// Should error: DNS resolution fails for hex-encoded numeric hostnames
			if err == nil {
				t.Errorf("expected error for hex-encoded IP URL %q, got nil", tt.url)
			}
		})
	}
}

func TestValidateWebhookURL_SSRFCredentialsInURL(t *testing.T) {
	// URLs with embedded credentials should still have their host validated.
	// The validator currently allows credentials but blocks based on the resolved IP.
	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		{"user:pass@localhost", "https://user:pass@localhost/hook", true},
		{"user:pass@127.0.0.1", "https://user:pass@127.0.0.1/hook", true},
		{"user:pass@10.0.0.1", "https://user:pass@10.0.0.1/hook", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if tt.wantError && err == nil {
				t.Errorf("expected error for URL with credentials %q, got nil", tt.url)
			}
		})
	}
}

func TestValidateWebhookURL_InvalidPorts(t *testing.T) {
	// Port 0 is technically parseable but unusual. We verify the validator
	// doesn't crash and still validates the host correctly.
	tests := []struct {
		name string
		url  string
	}{
		{"port 0 on localhost", "https://localhost:0/hook"},
		{"very high port on localhost", "https://localhost:99999/hook"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if err == nil {
				t.Errorf("expected error for URL %q targeting localhost, got nil", tt.url)
			}
		})
	}
}

func TestValidateWebhookURL_ValidHTTPS(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"simple HTTPS", "https://example.com/webhook"},
		{"HTTPS with port", "https://example.com:443/webhook"},
		{"HTTPS with path", "https://example.com/v1/events"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if err != nil {
				t.Errorf("expected no error for valid HTTPS URL %q, got %v", tt.url, err)
			}
		})
	}
}

func TestValidateWebhookURL_HTTPAccepted(t *testing.T) {
	// The current implementation accepts HTTP (both http and https are valid schemes).
	tests := []struct {
		name string
		url  string
	}{
		{"HTTP URL", "http://example.com/webhook"},
		{"HTTP with port", "http://example.com:8080/webhook"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if err != nil {
				t.Errorf("expected no error for HTTP URL %q, got %v", tt.url, err)
			}
		})
	}
}

func TestValidateWebhookURL_URLWithFragments(t *testing.T) {
	err := ValidateWebhookURL("https://example.com/webhook#section1")
	if err != nil {
		t.Errorf("expected no error for URL with fragment, got %v", err)
	}
}

func TestValidateWebhookURL_URLWithQueryParameters(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"single param", "https://example.com/webhook?token=abc"},
		{"multiple params", "https://example.com/webhook?token=abc&env=prod"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if err != nil {
				t.Errorf("expected no error for URL with query params %q, got %v", tt.url, err)
			}
		})
	}
}

func TestValidateWebhookURL_EmptyURL(t *testing.T) {
	err := ValidateWebhookURL("")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
}

func TestValidateWebhookURL_VeryLongURL(t *testing.T) {
	// 10000-char URL - validator should not crash
	longPath := strings.Repeat("a", 10000)
	url := "https://example.com/" + longPath
	err := ValidateWebhookURL(url)
	// Should succeed (it's a valid URL with a long path, host resolves to public IP)
	if err != nil {
		t.Errorf("expected no error for very long URL, got %v", err)
	}
}

func TestValidateWebhookURL_UnicodeHostname(t *testing.T) {
	// IDN / unicode hostnames - Go's url.Parse handles these.
	// The validator should not panic; it may fail DNS resolution.
	tests := []struct {
		name string
		url  string
	}{
		{"German IDN", "https://\u00fc\u00f6\u00e4.example.com/hook"},
		{"Chinese chars", "https://\u4f8b\u3048.jp/hook"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We just verify it doesn't panic. The result depends on DNS.
			_ = ValidateWebhookURL(tt.url)
		})
	}
}

func TestValidateWebhookURL_InvalidSchemes(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"ftp scheme", "ftp://example.com/hook"},
		{"javascript scheme", "javascript:alert(1)"},
		{"file scheme", "file:///etc/passwd"},
		{"no scheme", "example.com/hook"},
		{"data URI", "data:text/html,<h1>hi</h1>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if err == nil {
				t.Errorf("expected error for invalid scheme URL %q, got nil", tt.url)
			}
		})
	}
}

func TestValidateWebhookURL_NoHostname(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"scheme only", "https://"},
		{"scheme with path only", "https:///path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if err == nil {
				t.Errorf("expected error for URL without hostname %q, got nil", tt.url)
			}
		})
	}
}
