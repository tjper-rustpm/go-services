package rest

import (
	"encoding/json"
	"net/http"

	"github.com/tjper/rustcron/cmd/payment/model"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type Servers struct{ API }

func (ep Servers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	servers := make(model.Servers, 0)

	if !ep.checkoutEnabled {
		if err := json.NewEncoder(w).Encode(servers); err != nil {
			ihttp.ErrInternal(ep.logger, w, err)
		}
		return
	}

	servers, err := ep.store.FindServers(r.Context())
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := json.NewEncoder(w).Encode(servers); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
	}
}
