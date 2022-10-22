package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	ihttp "github.com/tjper/rustcron/internal/http"
	"go.uber.org/zap"
)

type CreateServer struct{ API }

func (ep CreateServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b CreateServerBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	if err := b.validateOwnerAndModeratorIntersection(); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	id, err := uuid.NewRandom()
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)

	var resp CreateServerResponse
	resp.FromUUID(id)

	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		ep.logger.Error("while encoding create server response", zap.Error(err))
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()

		if _, err := ep.ctrl.CreateServer(ctx, b.ToModelServer(id)); err != nil {
			ep.logger.Error("while creating server", zap.Error(err))
			return
		}
	}()
}
