package session

import (
	"context"
)

// ctxkey is a key used to store and retrieve a session from the context.
type ctxkey string

var sessionCtxKey ctxkey = "session_context_key"

// WithSession adds the session to the passed context.
func WithSession(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, sessionCtxKey, sess)
}

// FromContext retrieves the a Session specific to the current process from
// the passed context. The second return value indicates if the Session exists
// on the passed context. In order for FromContext to function as expected, the
// process calling FromContext, must be downstream of Middleware.
func FromContext(ctx context.Context) (*Session, bool) {
	sess, ok := ctx.Value(sessionCtxKey).(*Session)
	if !ok {
		return nil, false
	}
	return sess, ok
}
