package rest

import (
	errors "errors"
	http "net/http"

	"github.com/tjper/rustcron/cmd/user/controller"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type ChangePassword struct{ API }

func (ep ChangePassword) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Hash     string
		Password string
	}

	var b body
	if err := ep.read(w, r, &b); err != nil {
		return
	}

	err := ep.ctrl.ResetPassword(r.Context(), b.Hash, b.Password)
	if errors.Is(err, controller.ErrResetHashNotRecognized) ||
		errors.Is(err, controller.ErrPasswordResetRequestStale) {
		ihttp.ErrForbidden(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(w)
		return
	}

	ep.write(w, http.StatusCreated, nil)

}
