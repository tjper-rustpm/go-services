package rest

import (
	errors "errors"
	http "net/http"

	"github.com/go-chi/chi"
	"github.com/tjper/rustcron/cmd/user/controller"
	uerrors "github.com/tjper/rustcron/cmd/user/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type CreateUser struct{ API }

func (ep CreateUser) Route(router chi.Router) {
	router.Post("/user", ep.ServeHTTP)
}

func (ep CreateUser) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Email    string
		Password string
	}

	var b body
	if err := ep.read(w, r, b); err != nil {
		return
	}

	user, err := ep.ctrl.CreateUser(
		r.Context(),
		controller.CreateUserInput{Email: b.Email, Password: b.Password},
	)
	if emailErr := uerrors.AsEmailError(err); emailErr != nil {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}
	if passwordErr := uerrors.AsPasswordError(err); passwordErr != nil {
		http.Error(w, "invalid password", http.StatusBadRequest)
		return
	}
	if errors.Is(err, uerrors.EmailAlreadyInUse) {
		http.Error(w, "invalid email", http.StatusConflict)
		return
	}
	if err != nil {
		ihttp.ErrInternal(w)
		return
	}

	ep.write(w, http.StatusCreated, user)
}