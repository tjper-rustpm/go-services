package rest

import (
	http "net/http"

	"github.com/go-chi/chi"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type LogoutUser struct{ API }

func (ep LogoutUser) Route(router chi.Router) {
	router.Post("/logout", ep.ServeHTTP)
}

func (ep LogoutUser) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sess, ok := ep.session(r.Context(), w)
	if !ok {
		return
	}

	if err := ep.ctrl.LogoutUser(r.Context(), sess.ID); err != nil {
		ihttp.ErrInternal(w)
		return
	}

	ep.write(w, http.StatusCreated, nil)
}