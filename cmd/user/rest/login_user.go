package rest

import (
	http "net/http"
	"time"

	"github.com/tjper/rustcron/cmd/user/controller"
	uerrors "github.com/tjper/rustcron/cmd/user/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/rand"
	"github.com/tjper/rustcron/internal/session"
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

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
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

	sessionID, err := rand.GenerateString(32)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	sess := session.New(
		sessionID,
		user.ToSessionUser(),
		7*24*time.Hour,
	)

	if err := ep.sessionManager.CreateSession(r.Context(), *sess); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ihttp.SetSessionCookie(
		w,
		sessionID,
		ep.cookieOptions,
	)

	ep.write(w, http.StatusCreated, user)
}
