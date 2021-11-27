package rest

import (
	http "net/http"

	"github.com/tjper/rustcron/cmd/user/controller"
	uerrors "github.com/tjper/rustcron/cmd/user/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type LoginUser struct{ API }

func (ep LoginUser) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Email    string
		Password string
	}

	var b body
	if err := ep.read(w, r, &b); err != nil {
		return
	}

	user, err := ep.ctrl.LoginUser(
		r.Context(),
		controller.LoginUserInput{Email: b.Email, Password: b.Password},
	)
	if authErr := uerrors.AsAuthError(err); authErr != nil {
		ihttp.ErrUnauthorized(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ihttp.SetSessionCookie(
		w,
		user.SessionID,
		ep.cookieOptions,
	)

	ep.write(w, http.StatusCreated, user)
}
