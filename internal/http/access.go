package http

import (
	"context"
	"net/http"
)

func NewAccess(w http.ResponseWriter, r *http.Request) *Access {
	return &Access{
		w: w,
		r: r,
	}
}

// AccessMiddleware creates an Access instance and stores it in the context for
// each request processed by the middleware.
func AccessMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			access := NewAccess(w, r)
			ctx := WithAccess(r.Context(), access)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

type key string

var accessCtxKey key = "access_context_key"

// WithAccess stores an http.Access instance in the passed context. It may then
// be retrieve later with the AccessFromContext function.
func WithAccess(ctx context.Context, access *Access) context.Context {
	return context.WithValue(ctx, accessCtxKey, access)
}

// AccessFromContext may be used to retrieve http.Access from the processes
// context, if it is available.
func AccessFromContext(ctx context.Context) (*Access, bool) {
	access, ok := ctx.Value(accessCtxKey).(*Access)

	return access, ok
}

// Access wraps http related types and makes standard use-cases accessible to
// callers.
type Access struct {
	w http.ResponseWriter
	r *http.Request
}

const sessionKey = "_rpm-session"

// SessionID retrieves the session ID from the http Access instance.
func (a Access) SessionID() (string, bool) {
	cookie, err := a.r.Cookie(sessionKey)
	if err != nil {
		return "", false
	}

	return cookie.Value, true
}

// SetSessionID sets the session ID on the client via the underlying Access
// instance.
func (a Access) SetSessionID(
	sessionID,
	domain string,
	secure bool,
	sameSite http.SameSite,
) {
	http.SetCookie(a.w, &http.Cookie{
		Name:     sessionKey,
		Value:    sessionID,
		Domain:   domain,
		Path:     "/",
		Secure:   secure,
		HttpOnly: true,
		SameSite: sameSite,
	})
}
