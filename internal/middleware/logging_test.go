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

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
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

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/missing", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	logOutput := buf.String()
	if !bytes.Contains([]byte(logOutput), []byte("404")) {
		t.Errorf("log output missing status 404, got: %s", logOutput)
	}
}

func TestLoggingImplicitOKOnWrite(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Write body without calling WriteHeader — should default to 200
		_, _ = w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/implicit", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	logOutput := buf.String()
	if !bytes.Contains([]byte(logOutput), []byte("200")) {
		t.Errorf("expected status 200 in log, got: %s", logOutput)
	}
}

func TestLoggingDoubleWriteHeader(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		// Second WriteHeader should be ignored by statusWriter
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/double", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	logOutput := buf.String()
	if !bytes.Contains([]byte(logOutput), []byte("201")) {
		t.Errorf("expected status 201 (first call wins), got: %s", logOutput)
	}
}
