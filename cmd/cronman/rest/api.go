package rest

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/tjper/rustcron/cmd/cronman/model"
	ihttp "github.com/tjper/rustcron/internal/http"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IController interface {
	CreateServer(context.Context, model.ServerDefinition) (*model.DormantServer, error)
	ArchiveServer(context.Context, uuid.UUID) (*model.ArchivedServer, error)
	StartServer(context.Context, uuid.UUID) (*model.DormantServer, error)
	MakeServerLive(context.Context, uuid.UUID) (*model.LiveServer, error)
	StopServer(context.Context, uuid.UUID) (*model.DormantServer, error)

	ListServers(context.Context, interface{}) error
}

type API struct {
	logger *zap.Logger
	ctrl   IController
}

func (api API) CreateServer(w http.ResponseWriter, req *http.Request) {
	var b CreateServerBody
	if err := json.NewDecoder(req.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(w)
		return
	}

	sd, err := b.ToModelServerDefinition()
	if err != nil {
		ihttp.ErrInternal(w)
		return
	}

	server, err := api.ctrl.CreateServer(req.Context(), *sd)
	if err != nil {
		ihttp.ErrInternal(w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(server); err != nil {
		ihttp.ErrInternal(w)
		return
	}
}

func (api API) ArchiveServer() http.HandlerFunc {
	type body struct {
		ServerID uuid.UUID
	}

	return func(w http.ResponseWriter, req *http.Request) {
		var b body
		if err := json.NewDecoder(req.Body).Decode(&b); err != nil {
			ihttp.ErrInternal(w)
			return
		}

		server, err := api.ctrl.ArchiveServer(req.Context(), b.ServerID)
		if err != nil {
			ihttp.ErrInternal(w)
			return
		}

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(server); err != nil {
			ihttp.ErrInternal(w)
			return
		}
	}
}

func (api API) StartServer() http.HandlerFunc {
	type body struct {
		ServerID uuid.UUID
	}

	return func(w http.ResponseWriter, req *http.Request) {
		var b body
		if err := json.NewDecoder(req.Body).Decode(&b); err != nil {
			ihttp.ErrInternal(w)
			return
		}

		if _, err := api.ctrl.StartServer(req.Context(), b.ServerID); err != nil {
			ihttp.ErrInternal(w)
			return
		}

		liveServer, err := api.ctrl.MakeServerLive(req.Context(), b.ServerID)
		if err != nil {
			ihttp.ErrInternal(w)
			return
		}

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(liveServer); err != nil {
			ihttp.ErrInternal(w)
			return
		}
	}
}

func (api API) StopServer() http.HandlerFunc {
	type body struct {
		ServerID uuid.UUID
	}

	return func(w http.ResponseWriter, req *http.Request) {
		var b body
		if err := json.NewDecoder(req.Body).Decode(&b); err != nil {
			ihttp.ErrInternal(w)
			return
		}

		dormantServer, err := api.ctrl.StopServer(req.Context(), b.ServerID)
		if err != nil {
			ihttp.ErrInternal(w)
			return
		}

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(dormantServer); err != nil {
			ihttp.ErrInternal(w)
			return
		}
	}
}

func (api API) Servers() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {

		liveServers := make([]model.LiveServer, 0)
		if err := api.ctrl.ListServers(req.Context(), &liveServers); err != nil {
			ihttp.ErrInternal(w)
			return
		}

		dormantServers := make([]model.DormantServer, 0)
		if err := api.ctrl.ListServers(req.Context(), &dormantServers); err != nil {
			ihttp.ErrInternal(w)
			return
		}

		enc := json.NewEncoder(w)
		enc.Encode(liveServers)
		enc.Encode(dormantServers)
	}
}
