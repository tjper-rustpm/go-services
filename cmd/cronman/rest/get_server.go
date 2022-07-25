package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	ierrors "github.com/tjper/rustcron/cmd/cronman/errors"
	"github.com/tjper/rustcron/cmd/cronman/model"
	ihttp "github.com/tjper/rustcron/internal/http"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var errNoServerID = errors.New("missing server ID")

const serverIDParam = "serverID"

type GetServer struct{ API }

func (ep GetServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, serverIDParam)
	if serverID == "" {
		ihttp.ErrBadRequest(ep.logger, w, errNoServerID)
		return
	}

	id, err := uuid.Parse(serverID)
	if err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	serverI, err := ep.ctrl.GetServer(r.Context(), id)
	if errors.Is(err, ierrors.ErrServerDNE) {
		ihttp.ErrNotFound(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	var resp interface{}
	switch server := serverI.(type) {
	case *model.LiveServer:
		resp, err = LiveServerFromModel(*server)
	case *model.DormantServer:
		resp, err = DormantServerFromModel(*server)
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		ep.logger.Error("while encoding get server json", zap.Error(err))
		return
	}
}
