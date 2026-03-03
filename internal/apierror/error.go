package apierror

import (
	"encoding/json"
	"net/http"

	"github.com/manimovassagh/rampart/internal/middleware"
)

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiErr)
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
