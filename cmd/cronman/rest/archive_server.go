package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type ArchiveServer struct{ API }

func (ep ArchiveServer) Route(router chi.Router) {
	router.Post("/server/archive", ep.ServeHTTP)
}

func (ep ArchiveServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ServerID uuid.UUID
	}

	var b body
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(w)
		return
	}

	server, err := ep.ctrl.ArchiveServer(r.Context(), b.ServerID)
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
