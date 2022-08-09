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

type StartServer struct{ API }

func (ep StartServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	if _, ok := server.(*model.DormantServer); !ok {
		ihttp.ErrConflict(w)
		return
	}

	w.WriteHeader(http.StatusAccepted)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if _, err = ep.ctrl.StartServer(ctx, b.ServerID); err != nil {
			ep.logger.Error("while starting server", zap.Error(err))
			return
		}

		if _, err := ep.ctrl.MakeServerLive(ctx, b.ServerID); err != nil {
			ep.logger.Error("while making server live", zap.Error(err))
			return
		}
	}()
}
