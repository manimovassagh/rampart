package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggingWritesLog(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if buf.Len() == 0 {
		t.Error("expected log output, got nothing")
	}

	logOutput := buf.String()
	if !bytes.Contains([]byte(logOutput), []byte("/healthz")) {
		t.Errorf("log output missing path, got: %s", logOutput)
	}
}

func TestLoggingCapturesStatus(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	logOutput := buf.String()
	if !bytes.Contains([]byte(logOutput), []byte("404")) {
		t.Errorf("log output missing status 404, got: %s", logOutput)
	}
}
