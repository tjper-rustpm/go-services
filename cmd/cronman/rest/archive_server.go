package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	cronmanerrors "github.com/tjper/rustcron/cmd/cronman/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type ArchiveServer struct{ API }

func (ep ArchiveServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ServerID uuid.UUID `validate:"required"`
	}

	var b body
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	server, err := ep.ctrl.ArchiveServer(r.Context(), b.ServerID)
	if errors.Is(err, cronmanerrors.ErrServerDNE) {
		ihttp.ErrNotFound(w)
		return
	}
	if errors.Is(err, cronmanerrors.ErrServerNotDormant) {
		ihttp.ErrConflict(w)
		return
	}

	w.WriteHeader(http.StatusCreated)

	archived := ArchivedServerFromModel(*server)

	if err := json.NewEncoder(w).Encode(archived); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}
}
