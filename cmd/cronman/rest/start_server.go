package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	cronmanerrors "github.com/tjper/rustcron/cmd/cronman/errors"
	ihttp "github.com/tjper/rustcron/internal/http"

	"github.com/google/uuid"
)

type StartServer struct{ API }

func (ep StartServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ServerID uuid.UUID
	}

	var b body
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	_, err := ep.ctrl.StartServer(r.Context(), b.ServerID)
	if errors.Is(err, cronmanerrors.ErrServerDNE) {
		ihttp.ErrConflict(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	server, err := ep.ctrl.MakeServerLive(r.Context(), b.ServerID)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)

	live := LiveServerFromModel(*server)
	if err := json.NewEncoder(w).Encode(live); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}
}
