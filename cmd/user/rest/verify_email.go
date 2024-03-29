package rest

import (
	http "net/http"

	"github.com/go-chi/chi"
	uerrors "github.com/tjper/rustcron/cmd/user/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type VerifyEmail struct{ API }

func (ep VerifyEmail) Route(router chi.Router) {
	router.Post("user/verify-email", ep.ServeHTTP)
}

func (ep VerifyEmail) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Hash string `json:"hash" validate:"required"`
	}

	var b body
	if err := ep.read(w, r, &b); err != nil {
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	_, err := ep.ctrl.VerifyEmail(r.Context(), b.Hash)
	if authErr := uerrors.AsAuthError(err); authErr != nil {
		ihttp.ErrForbidden(w)
		return
	}
	if hashErr := uerrors.AsHashError(err); hashErr != nil {
		ihttp.ErrForbidden(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ep.write(w, http.StatusCreated, nil)
}
