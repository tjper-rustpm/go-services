package rest

import (
	"context"

	"github.com/go-chi/chi"
	"github.com/tjper/rustcron/cmd/cronman/model"

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

type Router interface {
	Route(chi.Router)
}

func NewAPI(
	logger *zap.Logger,
	ctrl IController,
) *API {
	api := API{
		Mux:    chi.NewRouter(),
		logger: logger,
		ctrl:   ctrl,
	}

	v1 := []Router{
		CreateServer{API: api},
		ArchiveServer{API: api},
		StartServer{API: api},
		StopServer{API: api},
		Servers{API: api},
	}

	api.Mux.Route("/v1", func(router chi.Router) {
		for _, endpoint := range v1 {
			endpoint.Route(router)
		}
	})

	return &api
}

type API struct {
	Mux *chi.Mux

	logger *zap.Logger
	ctrl   IController
}
