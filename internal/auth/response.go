package auth

import (
	"net/http"
	"time"
)

// setRefreshCookie sets the HttpOnly refresh-token cookie scoped to /auth/refresh.
func setRefreshCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		HttpOnly: true,
		Secure:   true,
		Path:     "/auth/refresh",
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
	})
}

// clearRefreshCookie expires the refresh-token cookie, instructing the browser to delete it.
func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		Path:     "/auth/refresh",
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
