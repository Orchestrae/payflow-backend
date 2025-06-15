// internal/api/response/response.go
package response

import (
	"encoding/json"
	"net/http"
	"payflow/internal/domain"
)

type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// RespondWithJSON sends a JSON response.
func RespondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "error marshalling response"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

// RespondWithError maps domain errors to HTTP status codes and sends an error response.
func RespondWithError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	resp := ErrorResponse{Error: "An unexpected error occurred"}

	switch err {
	case domain.ErrNotFound:
		status = http.StatusNotFound
		resp.Error = err.Error()
	case domain.ErrUnauthorized:
		status = http.StatusUnauthorized
		resp.Error = err.Error()
	case domain.ErrForbidden:
		status = http.StatusForbidden
		resp.Error = err.Error()
	case domain.ErrConflict:
		status = http.StatusConflict
		resp.Error = err.Error()
	case domain.ErrValidationFailed:
		status = http.StatusBadRequest
		resp.Error = err.Error()
	// Add more specific domain error mappings here
	default:
		// Log the full internal error for debugging, but don't expose it to the client.
		// log.Error().Err(err).Msg("Responding with internal server error")
	}

	RespondWithJSON(w, status, resp)
}
