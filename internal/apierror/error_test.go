package apierror

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func assertStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("status = %d, want %d", got, want)
	}
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) Error {
	t.Helper()
	var e Error
	if err := json.NewDecoder(w.Body).Decode(&e); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return e
}

func TestWriteBasicError(t *testing.T) {
	w := httptest.NewRecorder()
	Write(w, http.StatusBadRequest, "invalid_request", "Missing required field.")

	assertStatus(t, w.Code, http.StatusBadRequest)
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	got := decodeError(t, w)
	if got.Code != "invalid_request" {
		t.Errorf("error = %q, want invalid_request", got.Code)
	}
	if got.Description != "Missing required field." {
		t.Errorf("error_description = %q, want 'Missing required field.'", got.Description)
	}
	if got.Status != 400 {
		t.Errorf("body status = %d, want 400", got.Status)
	}
}

func TestWriteIncludesRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	w.Header().Set("X-Request-Id", "req_test123")

	Write(w, http.StatusInternalServerError, "internal_error", "Something broke.")

	got := decodeError(t, w)
	if got.RequestID != "req_test123" {
		t.Errorf("request_id = %q, want req_test123", got.RequestID)
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w)

	assertStatus(t, w.Code, http.StatusNotFound)
	got := decodeError(t, w)
	if got.Code != "not_found" {
		t.Errorf("error = %q, want not_found", got.Code)
	}
}

func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	InternalError(w)
	assertStatus(t, w.Code, http.StatusInternalServerError)
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	BadRequest(w, "email is required")

	got := decodeError(t, w)
	if got.Description != "email is required" {
		t.Errorf("description = %q, want 'email is required'", got.Description)
	}
}

func TestServiceUnavailable(t *testing.T) {
	w := httptest.NewRecorder()
	ServiceUnavailable(w, "database is down")
	assertStatus(t, w.Code, http.StatusServiceUnavailable)
}

func TestErrorMethod(t *testing.T) {
	e := &Error{Description: "test error"}
	if e.Error() != "test error" {
		t.Errorf("Error() = %q, want 'test error'", e.Error())
	}
}
