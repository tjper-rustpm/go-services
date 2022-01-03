package rest

import (
	"encoding/json"
	"net/http"

	"github.com/tjper/rustcron/cmd/cronman/model"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type Servers struct{ API }

func (ep Servers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	liveServers := make([]model.LiveServer, 0)
	if err := ep.ctrl.ListServers(r.Context(), &liveServers); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	dormantServers := make([]model.DormantServer, 0)
	if err := ep.ctrl.ListServers(r.Context(), &dormantServers); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	servers := make([]interface{}, 0, len(liveServers)+len(dormantServers))
	for _, server := range liveServers {
		servers = append(servers, LiveServerFromModel(server))
	}
	for _, server := range dormantServers {
		servers = append(servers, DormantServerFromModel(server))
	}

	if err := json.NewEncoder(w).Encode(servers); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}
}
