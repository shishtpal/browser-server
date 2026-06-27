package helpers

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the standard JSON error envelope returned by every API
// handler. Field-level validation problems are reported via Fields (keyed by
// the offending JSON field name) so clients can highlight individual inputs.
type ErrorResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

// WriteError writes a JSON error response with the given status code. Use this
// instead of http.Error so every error has the same shape and Content-Type.
func WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// WriteValidationError writes a 400 response describing one or more invalid
// fields. It is a no-op caller's responsibility to ensure fields is non-empty.
func WriteValidationError(w http.ResponseWriter, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(ErrorResponse{Error: "validation failed", Fields: fields})
}

// WriteJSON writes a JSON success response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
