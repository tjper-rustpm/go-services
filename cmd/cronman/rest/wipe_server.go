package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	ierrors "github.com/tjper/rustcron/cmd/cronman/errors"
	"github.com/tjper/rustcron/cmd/cronman/mapgen"
	"github.com/tjper/rustcron/cmd/cronman/model"
	ihttp "github.com/tjper/rustcron/internal/http"
	"go.uber.org/zap"

	"github.com/google/uuid"
)

type WipeServer struct{ API }

func (ep WipeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ServerID uuid.UUID      `validate:"required"`
		Kind     model.WipeKind `validate:"required"`
		Seed     uint32
		Salt     uint32
	}

	var b body
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	serverI, err := ep.ctrl.GetServer(r.Context(), b.ServerID)
	if errors.Is(err, ierrors.ErrServerDNE) {
		ihttp.ErrNotFound(w)
		return
	}

	var mapSize model.MapSizeKind
	switch server := serverI.(type) {
	case *model.LiveServer:
		mapSize = server.Server.MapSize
	case *model.DormantServer:
		mapSize = server.Server.MapSize
	}

	seed := mapgen.GenerateSeed(mapSize)
	if b.Seed != 0 {
		seed = b.Seed
	}

	salt := mapgen.GenerateSalt()
	if b.Salt != 0 {
		salt = b.Salt
	}

	wipe := model.Wipe{
		Kind:    b.Kind,
		MapSeed: seed,
		MapSalt: salt,
	}

	_, isDormant := serverI.(*model.DormantServer)

	if isDormant {
		if err := ep.ctrl.WipeServer(r.Context(), b.ServerID, wipe); err != nil {
			ihttp.ErrInternal(ep.logger, w, err)
			return
		}

		w.WriteHeader(http.StatusCreated)
		return
	}

	w.WriteHeader(http.StatusAccepted)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
		defer cancel()

		if _, err := ep.ctrl.StopServer(ctx, b.ServerID); err != nil {
			ihttp.ErrInternal(ep.logger, w, err)
			return
		}
		defer func() {
			if _, err := ep.ctrl.StartServer(ctx, b.ServerID); err != nil {
				ep.logger.Error("while restarting a wiped server", zap.Error(err))
				return
			}

			if _, err := ep.ctrl.MakeServerLive(ctx, b.ServerID); err != nil {
				ep.logger.Error("while make a wiped server live", zap.Error(err))
				return
			}
		}()

		if err := ep.ctrl.WipeServer(ctx, b.ServerID, wipe); err != nil {
			ihttp.ErrInternal(ep.logger, w, err)
			return
		}
	}()
}
