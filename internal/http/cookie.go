package http

import "net/http"

type CookieOptions struct {
	Domain   string
	Secure   bool
	SameSite http.SameSite
}

const sessionKey = "_rpm-session"

func SetSessionCookie(
	w http.ResponseWriter,
	id string,
	options CookieOptions,
) {
	http.SetCookie(
		w,
		&http.Cookie{
			Name:     sessionKey,
			Value:    id,
			Domain:   options.Domain,
			Path:     "/",
			Secure:   options.Secure,
			HttpOnly: true,
			SameSite: options.SameSite,
		},
	)
}

func SessionFromRequest(req *http.Request) string {
	cookie, err := req.Cookie(sessionKey)
	if err != nil {
		return ""
	}
	return cookie.Value
}
