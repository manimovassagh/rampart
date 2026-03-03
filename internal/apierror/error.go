package apierror

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/manimovassagh/rampart/internal/middleware"
)

// ContentTypeJSON is the standard JSON content type header value.
const ContentTypeJSON = "application/json"

// Error represents a consistent API error response.
// Format matches docs/api/overview.md.
type Error struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
	Status      int    `json:"status"`
	RequestID   string `json:"request_id,omitempty"`
}

func (e *Error) Error() string {
	return e.Description
}

// Write sends a JSON error response with the given status code.
// It reads X-Request-Id from the response header (set by requestid middleware).
func Write(w http.ResponseWriter, status int, code, description string) {
	reqID := w.Header().Get(middleware.HeaderRequestID)

	apiErr := &Error{
		Code:        code,
		Description: description,
		Status:      status,
		RequestID:   reqID,
	}

	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(apiErr); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}

// NotFound writes a 404 error response.
func NotFound(w http.ResponseWriter) {
	Write(w, http.StatusNotFound, "not_found", "The requested resource was not found.")
}

// InternalError writes a 500 error response.
func InternalError(w http.ResponseWriter) {
	Write(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
}

// BadRequest writes a 400 error response with a custom description.
func BadRequest(w http.ResponseWriter, description string) {
	Write(w, http.StatusBadRequest, "invalid_request", description)
}

// ServiceUnavailable writes a 503 error response.
func ServiceUnavailable(w http.ResponseWriter, description string) {
	Write(w, http.StatusServiceUnavailable, "service_unavailable", description)
}

// Conflict writes a 409 error response.
func Conflict(w http.ResponseWriter, description string) {
	Write(w, http.StatusConflict, "conflict", description)
}

// ValidationError is a 400 response with per-field errors.
type ValidationError struct {
	Code        string       `json:"error"`
	Description string       `json:"error_description"`
	Status      int          `json:"status"`
	RequestID   string       `json:"request_id,omitempty"`
	Fields      []FieldError `json:"fields"`
}

// FieldError identifies a validation problem with a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// WriteValidation sends a 400 response with field-level validation errors.
func WriteValidation(w http.ResponseWriter, fields []FieldError) {
	reqID := w.Header().Get(middleware.HeaderRequestID)

	ve := &ValidationError{
		Code:        "validation_error",
		Description: "One or more fields failed validation.",
		Status:      http.StatusBadRequest,
		RequestID:   reqID,
		Fields:      fields,
	}

	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(ve); err != nil {
		slog.Error("failed to encode validation error response", "error", err)
	}
}
