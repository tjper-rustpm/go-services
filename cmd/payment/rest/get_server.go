package rest

import (
	"encoding/json"
	"net/http"

	"github.com/tjper/rustcron/cmd/payment/model"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type Server struct{ API }

func (ep Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var servers model.Servers
	if err := ep.store.FindActiveSubscriptions(r.Context(), &servers); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	for _, server := range servers {
		ep.logger.Sugar().Infof("server: %v", server)
	}

	if err := json.NewEncoder(w).Encode(servers); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
	}
}
