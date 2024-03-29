package rest

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/controller"
	"github.com/tjper/rustcron/cmd/cronman/model"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
	"github.com/tjper/rustcron/internal/validator"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	validatorv10 "github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IController interface {
	CreateServer(context.Context, model.Server) (*model.DormantServer, error)
	GetServer(context.Context, uuid.UUID) (interface{}, error)
	UpdateServer(context.Context, controller.UpdateServerInput) (*model.DormantServer, error)
	ArchiveServer(context.Context, uuid.UUID) (*model.ArchivedServer, error)
	StartServer(context.Context, uuid.UUID) (*model.DormantServer, error)
	MakeServerLive(context.Context, uuid.UUID) (*model.LiveServer, error)
	StopServer(context.Context, uuid.UUID) (*model.DormantServer, error)
	WipeServer(context.Context, uuid.UUID, model.Wipe) error

	ListServers(context.Context, interface{}) error

	AddServerTags(context.Context, uuid.UUID, model.Tags) error
	RemoveServerTags(context.Context, uuid.UUID, []uuid.UUID) error

	AddServerEvents(context.Context, uuid.UUID, model.Events) error
	RemoveServerEvents(context.Context, uuid.UUID, []uuid.UUID) error

	AddServerModerators(context.Context, uuid.UUID, model.Moderators) error
	RemoveServerModerators(context.Context, uuid.UUID, []uuid.UUID) error

	AddServerOwners(context.Context, uuid.UUID, model.Owners) error
	RemoveServerOwners(context.Context, uuid.UUID, []uuid.UUID) error
}

type ISessionMiddleware interface {
	InjectSessionIntoCtx() func(http.Handler) http.Handler
	Touch() func(http.Handler) http.Handler
	HasRole(session.Role) func(http.Handler) http.Handler
}

func NewAPI(
	logger *zap.Logger,
	ctrl IController,
	sessionMiddleware ISessionMiddleware,
	healthz http.Handler,
) *API {
	api := API{
		Mux:    chi.NewRouter(),
		logger: logger,
		valid:  validator.New(),
		ctrl:   ctrl,
	}

	api.Mux.Handle("/healthz", healthz)

	api.Mux.Route("/v1", func(router chi.Router) {
		router.Use(
			sessionMiddleware.InjectSessionIntoCtx(),
			sessionMiddleware.Touch(),
			middleware.RequestLogger(ihttp.NewZapLogFormatter(logger)),
		)

		router.Group(func(router chi.Router) {
			router.Use(sessionMiddleware.HasRole(session.RoleAdmin))

			router.Method(http.MethodPatch, "/server", PatchServer{API: api})
			router.Method(http.MethodPost, "/server/archive", ArchiveServer{API: api})
			router.Method(http.MethodPost, "/server/wipe", WipeServer{API: api})

			router.Method(http.MethodPost, "/server/tags", AddServerTags{API: api})
			router.Method(http.MethodDelete, "/server/tags", RemoveServerTags{API: api})

			router.Method(http.MethodPost, "/server/events", AddServerEvents{API: api})
			router.Method(http.MethodDelete, "/server/events", RemoveServerEvents{API: api})

			router.Method(http.MethodPost, "/server/moderators", AddServerModerators{API: api})
			router.Method(http.MethodDelete, "/server/moderators", RemoveServerModerators{API: api})

			router.Method(http.MethodPost, "/server/owners", AddServerOwners{API: api})
			router.Method(http.MethodDelete, "/server/owners", RemoveServerOwners{API: api})

			router.Group(func(router chi.Router) {
				router.Use(middleware.Timeout(30 * time.Minute))

				router.Method(http.MethodPost, "/server/start", StartServer{API: api})
			})

			router.Group(func(router chi.Router) {
				router.Use(middleware.Timeout(10 * time.Minute))

				router.Method(http.MethodPost, "/server", CreateServer{API: api})
				router.Method(http.MethodPost, "/server/stop", StopServer{API: api})
			})
		})

		router.Method(http.MethodGet, "/servers", Servers{API: api})
		router.Method(http.MethodGet, fmt.Sprintf("/server/{%s}", serverIDParam), GetServer{API: api})
	})

	return &api
}

type API struct {
	Mux *chi.Mux

	logger *zap.Logger
	valid  *validatorv10.Validate
	ctrl   IController
}
