package rest

import (
	errors "errors"
	http "net/http"

	uerrors "github.com/tjper/rustcron/cmd/user/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type ForgotPassword struct{ API }

func (ep ForgotPassword) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Email string
	}

	var b body
	if err := ep.read(w, r, &b); err != nil {
		return
	}

	err := ep.ctrl.RequestPasswordReset(r.Context(), b.Email)
	if emailErr := uerrors.AsEmailError(err); emailErr != nil {
		ihttp.ErrBadRequest(w, "email")
		return
	}
	if errors.Is(err, uerrors.EmailAddressNotRecognized) {
		// Response is independent of whether an email address is found. This is
		// to prevent attackers from determining which email addresses are
		// associated with users.
		ep.write(w, http.StatusCreated, nil)
		return
	}
	if err != nil {
		ihttp.ErrInternal(w)
		return
	}

	ep.write(w, http.StatusCreated, nil)
}
