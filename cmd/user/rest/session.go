package rest

import (
	"context"
	errors "errors"
	"net/http"
	"time"

	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
)

type Session struct{ API }

func (ep Session) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sessionID := ihttp.SessionFromRequest(r)
	if sessionID == "" {
		ep.write(w, http.StatusNoContent, nil)
		return
	}

	sess, err := ep.sessionManager.RetrieveSession(r.Context(), sessionID)
	if errors.Is(err, session.ErrSessionDNE) {
		ep.write(w, http.StatusNoContent, nil)
		return
	}
	if errors.Is(err, session.ErrSessionStale) {
		sess, err = ep.refreshSession(r.Context(), *sess)
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ep.write(w, http.StatusOK, sess)
}

func (ep Session) refreshSession(
	ctx context.Context,
	sess session.Session,
) (*session.Session, error) {
	user, err := ep.ctrl.User(ctx, sess.User.ID)
	if err != nil {
		return nil, err
	}

	updateFn := func(sess *session.Session) {
		sess.User = user.ToSessionUser()
		sess.RefreshedAt = time.Now()
	}
	return ep.sessionManager.UpdateSession(
		ctx,
		sess.ID,
		updateFn,
	)
}
