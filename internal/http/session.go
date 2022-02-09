package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/tjper/rustcron/internal/session"
	"go.uber.org/zap"
)

// ISessionManager encompasses all manners by which a session may be interacted
// with.
type ISessionManager interface {
	RetrieveSession(context.Context, string) (*session.Session, error)
	TouchSession(context.Context, string, time.Duration) (*session.Session, error)
	DeleteSession(context.Context, session.Session) error
}

func NewSessionMiddleware(logger *zap.Logger, manager ISessionManager, expiration time.Duration) *SessionMiddleware {
	return &SessionMiddleware{
		logger:     logger,
		manager:    manager,
		expiration: expiration,
	}
}

type SessionMiddleware struct {
	logger  *zap.Logger
	manager ISessionManager

	expiration time.Duration
}

// InjectSessionIntoCtx injects the session associated with the request. If
// there is no session, the next handler is called. This middleware does not
// guarantee that a session exists within the request context.
func (sm SessionMiddleware) InjectSessionIntoCtx() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				var (
					sess *session.Session
					ctx  context.Context
					err  error
				)

				sessionID := SessionFromRequest(r)
				if sessionID == "" {
					goto serve
				}

				sess, err = sm.manager.RetrieveSession(r.Context(), sessionID)
				if errors.Is(err, session.ErrSessionDNE) {
					goto serve
				}
				if err != nil {
					ErrInternal(sm.logger, w, err)
					return
				}

				if sess.AbsoluteExpiration.Before(time.Now()) {
					if err := sm.manager.DeleteSession(r.Context(), *sess); err != nil {
						ErrInternal(sm.logger, w, err)
						return
					}
					goto serve
				}

				ctx = session.WithSession(r.Context(), sess)
				r = r.WithContext(ctx)

			serve:
				next.ServeHTTP(w, r)
			})
	}
}

// Touch updates the request's session with a timestamp indicating when
// activity last occurred. If session is not associated with the request, the
// next middleware is called.
func (sm SessionMiddleware) Touch() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				sess, ok := session.FromContext(r.Context())
				if !ok {
					goto serve
				}

				if _, err := sm.manager.TouchSession(r.Context(), sess.ID, sm.expiration); err != nil {
					ErrInternal(sm.logger, w, err)
					return
				}

			serve:
				next.ServeHTTP(w, r)
			})
	}
}

// HasRole ensures the request's session exists and has the role specified.
func (sm SessionMiddleware) HasRole(role session.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				sess, ok := session.FromContext(r.Context())
				if !ok {
					ErrUnauthorized(w)
					return
				}

				if sess.User.Role != role {
					ErrForbidden(w)
					return
				}

				next.ServeHTTP(w, r)
			})
	}
}

// IsAuthenticated ensures the request's session exists.
func (sm SessionMiddleware) IsAuthenticated() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				_, ok := session.FromContext(r.Context())
				if !ok {
					ErrUnauthorized(w)
					return
				}

				next.ServeHTTP(w, r)
			})
	}
}
