package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/model"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IController interface {
	CreateServer(context.Context, model.Server) (*model.DormantServer, error)
	UpdateServer(context.Context, model.DormantServer) (*model.DormantServer, error)
	ArchiveServer(context.Context, uuid.UUID) (*model.ArchivedServer, error)
	StartServer(context.Context, uuid.UUID) (*model.DormantServer, error)
	MakeServerLive(context.Context, uuid.UUID) (*model.LiveServer, error)
	StopServer(context.Context, uuid.UUID) (*model.DormantServer, error)

	ListServers(context.Context, interface{}) error
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

	api.Mux.Route("/v1", func(router chi.Router) {
		router.Group(func(router chi.Router) {
			router.Use(
				middleware.Timeout(30 * time.Minute),
			)

			router.Method(http.MethodPost, "/server/start", StartServer{API: api})
		})

		router.Group(func(router chi.Router) {
			router.Use(
				middleware.Timeout(10 * time.Minute),
			)

			router.Method(http.MethodPost, "/server", CreateServer{API: api})
			router.Method(http.MethodPost, "/server/stop", StopServer{API: api})
		})

		router.Method(http.MethodPut, "/server", PutServer{API: api})
		router.Method(http.MethodPost, "/server/archive", ArchiveServer{API: api})
		router.Method(http.MethodGet, "/servers", Servers{API: api})
	})

	return &api
}

type API struct {
	Mux *chi.Mux

	logger *zap.Logger
	ctrl   IController
}
