package rest

import (
	errors "errors"
	"net/http"

	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
)

type Me struct{ API }

func (ep Me) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sessionID := ihttp.SessionFromRequest(r)
	if sessionID == "" {
		ep.write(w, http.StatusNoContent, nil)
		return
	}

	sess, err := ep.sessionManager.RetrieveSession(r.Context(), sessionID)
	if errors.Is(err, session.ErrSessionDNE) {
		ep.write(w, http.StatusNoContent, nil)
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ep.write(w, http.StatusOK, sess.User)
}
