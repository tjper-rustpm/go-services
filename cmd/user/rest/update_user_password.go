package rest

import (
	http "net/http"

	"github.com/tjper/rustcron/cmd/user/controller"
	uerrors "github.com/tjper/rustcron/cmd/user/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
)

type UpdateUserPassword struct{ API }

func (ep UpdateUserPassword) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		CurrentPassword string `json:"currentPassword" validate:"required,password"`
		NewPassword     string `json:"newPassword" validate:"required,password"`
	}

	var b body
	if err := ep.read(w, r, &b); err != nil {
		return
	}

	ep.logger.Sugar().Infof("update user password; body: %v\n", b)

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	sess, ok := session.FromContext(r.Context())
	if !ok {
		ihttp.ErrUnauthorized(w)
		return
	}

	user, err := ep.ctrl.UpdateUserPassword(
		r.Context(),
		controller.UpdateUserPasswordInput{
			ID:              sess.User.ID,
			CurrentPassword: b.CurrentPassword,
			NewPassword:     b.NewPassword,
		},
	)
	if authErr := uerrors.AsAuthError(err); authErr != nil {
		ihttp.ErrForbidden(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.ctrl.LogoutAllUserSessions(r.Context(), sess.User.ID); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ep.write(w, http.StatusCreated, user)
}
