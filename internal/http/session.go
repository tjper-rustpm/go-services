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
	Retrieve(context.Context, string) (*session.Session, error)
	Touch(context.Context, string, time.Duration) error
	Delete(context.Context, string) error
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

				sess, err := manager.Retrieve(r.Context(), sessionID)
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
					if err := manager.Delete(r.Context(), sessionID); err != nil {
						logger.Error("error deleting session", zap.Error(err))
					}

					ErrUnauthorized(w)
					return
				}

				if err := manager.Touch(r.Context(), sessionID, exp); err != nil {
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
