package rest

import (
	http "net/http"
	"time"

	usererrors "github.com/tjper/rustcron/cmd/user/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type ResetPassword struct{ API }

func (ep ResetPassword) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Hash     string `json:"hash" validate:"required"`
		Password string `json:"password" validate:"required,password"`
	}

	var b body
	if err := ep.read(w, r, &b); err != nil {
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	user, err := ep.ctrl.ResetPassword(r.Context(), b.Hash, b.Password)
	if autherr := usererrors.AsAuthError(err); autherr != nil {
		ihttp.ErrForbidden(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.sessionManager.InvalidateUserSessionsBefore(
		r.Context(),
		user.ID,
		time.Now(),
	); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ep.write(w, http.StatusCreated, nil)
}
