package rest

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type DeleteServer struct{ API }

func (ep DeleteServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ServerID uuid.UUID
	}

	var b body
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	server, err := ep.ctrl.ArchiveServer(r.Context(), b.ServerID)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(server); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

}
