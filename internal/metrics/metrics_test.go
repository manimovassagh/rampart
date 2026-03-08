package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMiddlewareRecordsMetrics(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	// Verify counter was incremented
	count := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/healthz", "200"))
	if count < 1 {
		t.Errorf("HTTPRequestsTotal = %f, want >= 1", count)
	}
}

func TestMiddlewareCapturesStatusCode(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/missing", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	count := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/missing", "404"))
	if count < 1 {
		t.Errorf("HTTPRequestsTotal for 404 = %f, want >= 1", count)
	}
}

func TestHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody)
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "rampart_http_requests_total") {
		t.Error("metrics output missing rampart_http_requests_total")
	}
}

func TestAuthTotalCounter(t *testing.T) {
	AuthTotal.WithLabelValues("success").Inc()
	AuthTotal.WithLabelValues("failure").Inc()
	AuthTotal.WithLabelValues("failure").Inc()

	success := testutil.ToFloat64(AuthTotal.WithLabelValues("success"))
	if success < 1 {
		t.Errorf("auth success count = %f, want >= 1", success)
	}
	failure := testutil.ToFloat64(AuthTotal.WithLabelValues("failure"))
	if failure < 2 {
		t.Errorf("auth failure count = %f, want >= 2", failure)
	}
}

func TestTokensIssuedCounter(t *testing.T) {
	TokensIssued.WithLabelValues("access").Inc()
	TokensIssued.WithLabelValues("refresh").Inc()

	access := testutil.ToFloat64(TokensIssued.WithLabelValues("access"))
	if access < 1 {
		t.Errorf("access token count = %f, want >= 1", access)
	}
}

func TestActiveSessionsGauge(t *testing.T) {
	ActiveSessions.Set(42)
	val := testutil.ToFloat64(ActiveSessions)
	if val != 42 {
		t.Errorf("active sessions = %f, want 42", val)
	}
}
