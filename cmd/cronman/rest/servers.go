package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/tjper/rustcron/cmd/cronman/model"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type Servers struct{ API }

func (ep Servers) Route(router chi.Router) {
	router.Get("/servers", ep.ServeHTTP)
}

func (ep Servers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	liveServers := make([]model.LiveServer, 0)
	if err := ep.ctrl.ListServers(r.Context(), &liveServers); err != nil {
		ihttp.ErrInternal(w)
		return
	}

	dormantServers := make([]model.DormantServer, 0)
	if err := ep.ctrl.ListServers(r.Context(), &dormantServers); err != nil {
		ihttp.ErrInternal(w)
		return
	}

	enc := json.NewEncoder(w)
	enc.Encode(liveServers)
	enc.Encode(dormantServers)
}