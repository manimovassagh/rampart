package apierror

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWrite_BasicError(t *testing.T) {
	w := httptest.NewRecorder()
	Write(w, http.StatusBadRequest, "invalid_request", "Missing required field.")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var got Error
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.Code != "invalid_request" {
		t.Errorf("error = %q, want invalid_request", got.Code)
	}
	if got.Description != "Missing required field." {
		t.Errorf("error_description = %q, want 'Missing required field.'", got.Description)
	}
	if got.Status != 400 {
		t.Errorf("status = %d, want 400", got.Status)
	}
}

func TestWrite_IncludesRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	w.Header().Set("X-Request-Id", "req_test123")

	Write(w, http.StatusInternalServerError, "internal_error", "Something broke.")

	var got Error
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.RequestID != "req_test123" {
		t.Errorf("request_id = %q, want req_test123", got.RequestID)
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var got Error
	json.NewDecoder(w.Body).Decode(&got)
	if got.Code != "not_found" {
		t.Errorf("error = %q, want not_found", got.Code)
	}
}

func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	InternalError(w)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	BadRequest(w, "email is required")

	var got Error
	json.NewDecoder(w.Body).Decode(&got)
	if got.Description != "email is required" {
		t.Errorf("description = %q, want 'email is required'", got.Description)
	}
}

func TestServiceUnavailable(t *testing.T) {
	w := httptest.NewRecorder()
	ServiceUnavailable(w, "database is down")

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestError_ErrorMethod(t *testing.T) {
	e := &Error{Description: "test error"}
	if e.Error() != "test error" {
		t.Errorf("Error() = %q, want 'test error'", e.Error())
	}
}
