package rest

import (
	http "net/http"
	"time"

	ihttp "github.com/tjper/rustcron/internal/http"
)

type LogoutAllUser struct{ API }

func (ep LogoutAllUser) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sess, ok := ep.session(r.Context(), w)
	if !ok {
		return
	}

	if err := ep.sessionManager.InvalidateUserSessionsBefore(
		r.Context(),
		sess.User.ID,
		time.Now(),
	); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ep.write(w, http.StatusCreated, nil)
}
