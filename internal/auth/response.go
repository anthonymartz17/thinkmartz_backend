package auth

import (
	"net/http"
	"time"
)

// setRefreshCookie sets the HttpOnly refresh-token cookie scoped to
// /v1/auth, a common ancestor of /v1/auth/refresh and /v1/auth/logout —
// the two routes that read it.
func setRefreshCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		HttpOnly: true,
		Secure:   true,
		Path:     "/v1/auth",
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
	})
}

// clearRefreshCookie expires the refresh-token cookie, instructing the
// browser to delete it. Path must match setRefreshCookie's exactly, or the
// browser won't recognize it as the same cookie to clear.
func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		Path:     "/v1/auth",
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
