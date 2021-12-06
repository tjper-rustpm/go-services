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
	TouchSession(context.Context, session.Session, time.Duration) error
	DeleteSession(context.Context, session.Session) error
}

func Session(logger *zap.Logger, manager ISessionManager, exp time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				sessionID := SessionFromRequest(r)
				if sessionID == "" {
					ErrUnauthorized(w)
					return
				}

				sess, err := manager.RetrieveSession(r.Context(), sessionID)
				if errors.Is(err, session.ErrSessionDNE) {
					ErrUnauthorized(w)
					return
				}
				if err != nil {
					ErrInternal(logger, w, err)
					return
				}

				if sess.AbsoluteExpiration.Before(time.Now()) {
					logger.Info(
						"deleting base on absolute expiration",
						zap.Time("exp", sess.AbsoluteExpiration),
						zap.Time("now", time.Now()),
					)
					if err := manager.DeleteSession(r.Context(), *sess); err != nil {
						logger.Error("error deleting session", zap.Error(err))
					}

					ErrUnauthorized(w)
					return
				}

				if err := manager.TouchSession(r.Context(), *sess, exp); err != nil {
					ErrInternal(logger, w, err)
					return
				}

				ctx := session.WithSession(r.Context(), sess)
				req := r.WithContext(ctx)
				next.ServeHTTP(w, req)
			},
		)
	}
}
