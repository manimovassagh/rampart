package database

import (
	"testing"

	"github.com/manimovassagh/rampart/internal/model"
)

func TestValidateRedirectURI(t *testing.T) {
	tests := []struct {
		name     string
		client   *model.OAuthClient
		uri      string
		expected bool
	}{
		{
			name:     "exact match single URI",
			client:   &model.OAuthClient{RedirectURIs: []string{"http://localhost:3000/callback"}},
			uri:      "http://localhost:3000/callback",
			expected: true,
		},
		{
			name:     "exact match from multiple URIs",
			client:   &model.OAuthClient{RedirectURIs: []string{"https://a.com/cb", "https://b.com/cb"}},
			uri:      "https://b.com/cb",
			expected: true,
		},
		{
			name:     "no match",
			client:   &model.OAuthClient{RedirectURIs: []string{"https://a.com/cb"}},
			uri:      "https://evil.com/cb",
			expected: false,
		},
		{
			name:     "empty URI list",
			client:   &model.OAuthClient{RedirectURIs: []string{}},
			uri:      "https://a.com/cb",
			expected: false,
		},
		{
			name:     "nil URI list",
			client:   &model.OAuthClient{RedirectURIs: nil},
			uri:      "https://a.com/cb",
			expected: false,
		},
		{
			name:     "empty input URI",
			client:   &model.OAuthClient{RedirectURIs: []string{"https://a.com/cb"}},
			uri:      "",
			expected: false,
		},
		{
			name:     "partial match rejected (prefix)",
			client:   &model.OAuthClient{RedirectURIs: []string{"https://a.com/callback"}},
			uri:      "https://a.com/call",
			expected: false,
		},
		{
			name:     "trailing slash matters",
			client:   &model.OAuthClient{RedirectURIs: []string{"https://a.com/cb"}},
			uri:      "https://a.com/cb/",
			expected: false,
		},
		{
			name:     "case sensitive",
			client:   &model.OAuthClient{RedirectURIs: []string{"https://A.com/cb"}},
			uri:      "https://a.com/cb",
			expected: false,
		},
		{
			name:     "query string matters",
			client:   &model.OAuthClient{RedirectURIs: []string{"https://a.com/cb"}},
			uri:      "https://a.com/cb?foo=bar",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateRedirectURI(tc.client, tc.uri)
			if got != tc.expected {
				t.Errorf("ValidateRedirectURI(%q) = %v, want %v", tc.uri, got, tc.expected)
			}
		})
	}
}

func TestParseRedirectURIs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single URI",
			input:    "https://example.com/callback",
			expected: []string{"https://example.com/callback"},
		},
		{
			name:     "multiple URIs newline separated",
			input:    "https://a.com/cb\nhttps://b.com/cb",
			expected: []string{"https://a.com/cb", "https://b.com/cb"},
		},
		{
			name:     "trims whitespace",
			input:    "  https://a.com/cb  \n  https://b.com/cb  ",
			expected: []string{"https://a.com/cb", "https://b.com/cb"},
		},
		{
			name:     "skips empty lines",
			input:    "https://a.com/cb\n\n\nhttps://b.com/cb\n",
			expected: []string{"https://a.com/cb", "https://b.com/cb"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name:     "only whitespace",
			input:    "  \n  \n  ",
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseRedirectURIs(tc.input)
			if len(got) != len(tc.expected) {
				t.Fatalf("parseRedirectURIs(%q) returned %d URIs, want %d: %v", tc.input, len(got), len(tc.expected), got)
			}
			for i, uri := range got {
				if uri != tc.expected[i] {
					t.Errorf("parseRedirectURIs(%q)[%d] = %q, want %q", tc.input, i, uri, tc.expected[i])
				}
			}
		})
	}
}

func TestGenerateClientID(t *testing.T) {
	// Should produce non-empty, unique IDs
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateClientID()
		if id == "" {
			t.Fatal("generateClientID returned empty string")
		}
		if len(id) != 32 {
			t.Errorf("expected 32-char hex string, got %d chars: %q", len(id), id)
		}
		if seen[id] {
			t.Errorf("duplicate client ID generated: %q", id)
		}
		seen[id] = true
	}
}
