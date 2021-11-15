package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type StartServer struct{ API }

func (ep StartServer) Route(router chi.Router) {
	router.Post("/server/start", ep.ServeHTTP)
}

func (ep StartServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ServerID uuid.UUID
	}

	var b body
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(w)
		return
	}

	if _, err := ep.ctrl.StartServer(r.Context(), b.ServerID); err != nil {
		ihttp.ErrInternal(w)
		return
	}

	liveServer, err := ep.ctrl.MakeServerLive(r.Context(), b.ServerID)
	if err != nil {
		ihttp.ErrInternal(w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(liveServer); err != nil {
		ihttp.ErrInternal(w)
		return
	}
}
