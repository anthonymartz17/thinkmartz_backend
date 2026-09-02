package auth

import (
	"encoding/json"
	"net/http"
)

// writeJSON writes body as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, body any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(body)
}

// writeError writes a plain-text error response with the given status code.
func writeError(w http.ResponseWriter, status int, msg string) {
	http.Error(w, msg, status)
}

// setRefreshCookie sets the HttpOnly refresh-token cookie scoped to /auth/refresh.
func setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		HttpOnly: true,
		Secure:   true,
		Value:    token,
		Path:     "/auth/refresh",
		SameSite: http.SameSiteStrictMode,
	})
}
