package rest

import (
	"encoding/json"
	"net/http"

	ihttp "github.com/tjper/rustcron/internal/http"
)

type CreateServer struct{ API }

func (ep CreateServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b CreateServerBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	server, err := ep.ctrl.CreateServer(r.Context(), b.ToModelServer())
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)

	dormant := DormantServerFromModel(*server)

	if err := json.NewEncoder(w).Encode(dormant); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}
}
