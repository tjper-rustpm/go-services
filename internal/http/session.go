package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/tjper/rustcron/internal/session"
	"go.uber.org/zap"
)

// Retriever retrieves a Session for the current process.
type Retriever interface {
	Retrieve(context.Context, string) (*session.Session, error)
}

func SessionAuthenticated(logger *zap.Logger, retriever Retriever) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				sessionID := SessionFromRequest(r)
				if sessionID == "" {
					ErrUnauthorized(w)
					return
				}

				sess, err := retriever.Retrieve(r.Context(), sessionID)
				if errors.Is(err, session.ErrSessionDNE) {
					ErrUnauthorized(w)
					return
				}
				if err != nil {
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
