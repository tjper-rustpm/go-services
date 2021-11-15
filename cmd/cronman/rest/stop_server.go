package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type StopServer struct{ API }

func (ep StopServer) Route(router chi.Router) {
	router.Post("/server/stop", ep.ServeHTTP)
}

func (ep StopServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ServerID uuid.UUID
	}

	var b body
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(w)
		return
	}

	dormantServer, err := ep.ctrl.StopServer(r.Context(), b.ServerID)
	if err != nil {
		ihttp.ErrInternal(w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(dormantServer); err != nil {
		ihttp.ErrInternal(w)
		return
	}
}
