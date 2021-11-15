package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type CreateServer struct{ API }

func (ep CreateServer) Route(router chi.Router) {
	router.Post("/server", ep.ServeHTTP)
}

func (ep CreateServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b CreateServerBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(w)
		return
	}

	sd, err := b.ToModelServerDefinition()
	if err != nil {
		ihttp.ErrInternal(w)
		return
	}

	server, err := ep.ctrl.CreateServer(r.Context(), *sd)
	if err != nil {
		ihttp.ErrInternal(w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(server); err != nil {
		ihttp.ErrInternal(w)
		return
	}
}
