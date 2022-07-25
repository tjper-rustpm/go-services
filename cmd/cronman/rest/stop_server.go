package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	ierrors "github.com/tjper/rustcron/cmd/cronman/errors"
	"github.com/tjper/rustcron/cmd/cronman/model"
	ihttp "github.com/tjper/rustcron/internal/http"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type StopServer struct{ API }

func (ep StopServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	server, err := ep.ctrl.GetServer(r.Context(), b.ServerID)
	if errors.Is(err, ierrors.ErrServerDNE) {
		ihttp.ErrNotFound(w)
		return
	}

	if _, ok := server.(*model.LiveServer); !ok {
		ihttp.ErrConflict(w)
		return
	}

	w.WriteHeader(http.StatusAccepted)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if _, err := ep.ctrl.StopServer(ctx, b.ServerID); err != nil {
		ep.logger.Error("while stopping server", zap.Error(err))
		return
	}
}
