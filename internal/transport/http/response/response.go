// Package response provides shared HTTP response-writing helpers for domain handlers.
package response

import (
	"encoding/json"
	"net/http"
)

// WriteJSON writes body as a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, body any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(body)
}

// WriteError writes a plain-text error response with the given status code.
func WriteError(w http.ResponseWriter, status int, msg string) {
	http.Error(w, msg, status)
}
