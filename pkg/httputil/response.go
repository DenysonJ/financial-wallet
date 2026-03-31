package httputil

import (
	"encoding/json"
	"net/http"
)

// SuccessResponse is the standard envelope for successful API responses.
// Format: {"data": ..., "meta": ..., "links": ...}
type SuccessResponse struct {
	Data  any `json:"data"`
	Meta  any `json:"meta,omitempty"`
	Links any `json:"links,omitempty"`
}

// ErrorDetail holds error details for API error responses.
type ErrorDetail struct {
	Message string         `json:"message"`
	Code    string         `json:"code,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

// ErrorResponse is the standard envelope for API error responses.
// Format: {"errors": {"message": ..., "code": ..., "details": ...}}
type ErrorResponse struct {
	Errors ErrorDetail `json:"errors"`
}

// WriteSuccess writes a standardized success response using http.ResponseWriter.
func WriteSuccess(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, SuccessResponse{Data: data})
}

// WriteSuccessWithMeta writes a standardized success response with metadata and links.
func WriteSuccessWithMeta(w http.ResponseWriter, status int, data, meta, links any) {
	writeJSON(w, status, SuccessResponse{
		Data:  data,
		Meta:  meta,
		Links: links,
	})
}

// WriteError writes a standardized error response using http.ResponseWriter.
func WriteError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{
		Errors: ErrorDetail{Message: message},
	})
}

// WriteErrorWithCode writes a standardized error response with an error code.
func WriteErrorWithCode(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Errors: ErrorDetail{
			Message: message,
			Code:    code,
		},
	})
}

// WriteErrorWithDetails writes a standardized error response with code and details.
func WriteErrorWithDetails(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	writeJSON(w, status, ErrorResponse{
		Errors: ErrorDetail{
			Message: message,
			Code:    code,
			Details: details,
		},
	})
}

// writeJSON encodes v as JSON and writes it to the ResponseWriter with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
