package rest

import (
	errors "errors"
	http "net/http"

	"github.com/tjper/rustcron/cmd/user/controller"
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

	err := ep.ctrl.ResetPassword(r.Context(), b.Hash, b.Password)
	if errors.Is(err, controller.ErrResetHashNotRecognized) ||
		errors.Is(err, controller.ErrPasswordResetRequestStale) {
		ihttp.ErrForbidden(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ep.write(w, http.StatusCreated, nil)
}
