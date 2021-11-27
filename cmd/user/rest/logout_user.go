package rest

import (
	http "net/http"

	ihttp "github.com/tjper/rustcron/internal/http"
)

type LogoutUser struct{ API }

func (ep LogoutUser) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sess, ok := ep.session(r.Context(), w)
	if !ok {
		return
	}

	if err := ep.ctrl.LogoutUser(r.Context(), sess.ID); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ep.write(w, http.StatusCreated, nil)
}
