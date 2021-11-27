package rest

import (
	http "net/http"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/user/controller"
	uerrors "github.com/tjper/rustcron/cmd/user/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
)

type UpdateUserPassword struct{ API }

func (ep UpdateUserPassword) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		UserID          uuid.UUID
		CurrentPassword string
		NewPassword     string
	}

	var b body
	if err := ep.read(w, r, &b); err != nil {
		return
	}

	sess, ok := session.FromContext(r.Context())
	if !ok {
		ihttp.ErrUnauthorized(w)
		return
	}
	if !sess.IsAuthorized(b.UserID) {
		ihttp.ErrForbidden(w)
		return
	}

	user, err := ep.ctrl.UpdateUserPassword(
		r.Context(),
		controller.UpdateUserPasswordInput{
			ID:              b.UserID,
			CurrentPassword: b.CurrentPassword,
			NewPassword:     b.NewPassword,
		},
	)
	if passwordErr := uerrors.AsPasswordError(err); passwordErr != nil {
		ihttp.ErrBadRequest(w, "password")
		return
	}
	if authErr := uerrors.AsAuthError(err); authErr != nil {
		ihttp.ErrForbidden(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.ctrl.LogoutUser(r.Context(), sess.ID); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ep.write(w, http.StatusCreated, user)
}
