package rest

import (
	errors "errors"
	http "net/http"

	uerrors "github.com/tjper/rustcron/cmd/user/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type ResendEmailVerification struct{ API }

func (ep ResendEmailVerification) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sess, ok := ep.session(r.Context(), w)
	if !ok {
		return
	}

	_, err := ep.ctrl.ResendEmailVerification(r.Context(), sess.User.ID)
	if errors.Is(err, uerrors.ErrEmailAlreadyVerified) {
		ihttp.ErrConflict(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ep.write(w, http.StatusCreated, nil)
}
