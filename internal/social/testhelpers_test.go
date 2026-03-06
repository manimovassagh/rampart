package social

import (
	"net/http"
	"strings"
)

// redirectTransport redirects requests from hardcoded URLs to test server URLs.
type redirectTransport struct {
	urlMap    map[string]string
	transport http.RoundTripper
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	originalURL := req.URL.String()
	// Find the longest matching prefix to avoid ambiguity
	// (e.g., /user vs /user/emails).
	bestFrom := ""
	bestTo := ""
	for from, to := range rt.urlMap {
		if strings.HasPrefix(originalURL, from) && len(from) > len(bestFrom) {
			bestFrom = from
			bestTo = to
		}
	}
	if bestFrom != "" {
		newURL := bestTo + strings.TrimPrefix(originalURL, bestFrom)
		newReq := req.Clone(req.Context())
		parsed, err := req.URL.Parse(newURL)
		if err != nil {
			return nil, err
		}
		newReq.URL = parsed
		return rt.transport.RoundTrip(newReq)
	}
	return rt.transport.RoundTrip(req)
}

// newRedirectClient creates an HTTP client that redirects requests
// from production URLs to test server URLs.
func newRedirectClient(urlMap map[string]string) *http.Client {
	return &http.Client{
		Transport: &redirectTransport{
			urlMap:    urlMap,
			transport: http.DefaultTransport,
		},
	}
}
