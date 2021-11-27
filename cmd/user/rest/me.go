package rest

import (
	http "net/http"

	"github.com/go-chi/chi"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type Me struct{ API }

func (ep Me) Route(router chi.Router) {
	router.Get("/me", ep.ServeHTTP)
}

func (ep Me) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sess, ok := ep.session(r.Context(), w)
	if !ok {
		return
	}

	user, err := ep.ctrl.User(r.Context(), sess.User.ID)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	ep.write(w, http.StatusCreated, user)
}
