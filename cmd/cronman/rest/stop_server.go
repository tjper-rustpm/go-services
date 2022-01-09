package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	cronmanerrors "github.com/tjper/rustcron/cmd/cronman/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type StopServer struct{ API }

func (ep StopServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ServerID uuid.UUID
	}

	var b body
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	server, err := ep.ctrl.StopServer(r.Context(), b.ServerID)
	if errors.Is(err, cronmanerrors.ErrServerDNE) {
		ihttp.ErrConflict(w)
		return
	}
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
