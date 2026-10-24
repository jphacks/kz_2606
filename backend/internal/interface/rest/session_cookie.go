package rest

import (
	"net/http"
	"time"
)

const sessionCookieName = "session"

func writeSessionCookie(w http.ResponseWriter, token string, options CookieOptions) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   options.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(options.Lifetime.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter, options CookieOptions) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   options.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
}
