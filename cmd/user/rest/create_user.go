package rest

import (
	errors "errors"
	http "net/http"

	"github.com/tjper/rustcron/cmd/user/controller"
	uerrors "github.com/tjper/rustcron/cmd/user/errors"
	ihttp "github.com/tjper/rustcron/internal/http"

	"go.uber.org/zap"
)

type CreateUser struct{ API }

func (ep CreateUser) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Email    string `json:"email" validate:"required,email"`
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

	user, err := ep.ctrl.CreateUser(
		r.Context(),
		controller.CreateUserInput{Email: b.Email, Password: b.Password},
	)
	if errors.Is(err, uerrors.EmailAlreadyInUse) {
		ihttp.ErrConflict(w)
		return
	}
	if err != nil {
		ep.logger.Error("creating user", zap.Error(err))
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ep.write(w, http.StatusCreated, user)
}
