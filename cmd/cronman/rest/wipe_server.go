package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	ierrors "github.com/tjper/rustcron/cmd/cronman/errors"
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
		Seed     uint16
		Salt     uint16
	}

	var b body
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	server, err := ep.ctrl.GetServer(r.Context(), b.ServerID)
	if errors.Is(err, ierrors.ErrServerDNE) {
		ihttp.ErrNotFound(w)
		return
	}

	_, isLive := server.(*model.LiveServer)

	if isLive {
		if _, err := ep.ctrl.StopServer(r.Context(), b.ServerID); err != nil {
			ihttp.ErrInternal(ep.logger, w, err)
			return
		}
		defer func() {
			if _, err := ep.ctrl.StartServer(r.Context(), b.ServerID); err != nil {
				ep.logger.Error("while restarting a wiped server", zap.Error(err))
				return
			}

			if _, err := ep.ctrl.MakeServerLive(r.Context(), b.ServerID); err != nil {
				ep.logger.Error("while make a wiped server live", zap.Error(err))
				return
			}
		}()
	}

	seed := model.GenerateSeed()
	if b.Seed != 0 {
		seed = b.Seed
	}

	salt := model.GenerateSalt()
	if b.Salt != 0 {
		salt = b.Salt
	}

	wipe := model.Wipe{
		Kind:    b.Kind,
		MapSeed: seed,
		MapSalt: salt,
	}

	if err := ep.ctrl.WipeServer(r.Context(), b.ServerID, wipe); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
	}
}
