package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	cronmanerrors "github.com/tjper/rustcron/cmd/cronman/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type PutServer struct{ API }

func (ep PutServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b PutServerBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	server, err := ep.ctrl.UpdateServer(r.Context(), b.ToModelDormantServer())
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
