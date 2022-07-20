package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/controller"
	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/cmd/cronman/userdata"
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
	UpdateServer(context.Context, controller.UpdateServerInput) (*model.DormantServer, error)
	ArchiveServer(context.Context, uuid.UUID) (*model.ArchivedServer, error)
	StartServer(context.Context, uuid.UUID, ...userdata.Option) (*model.DormantServer, error)
	MakeServerLive(context.Context, uuid.UUID) (*model.LiveServer, error)
	StopServer(context.Context, uuid.UUID) (*model.DormantServer, error)

	ListServers(context.Context, interface{}) error

	AddServerTags(context.Context, uuid.UUID, model.Tags) error
	RemoveServerTags(context.Context, uuid.UUID, []uuid.UUID) error

	AddServerEvents(context.Context, uuid.UUID, model.Events) error
	RemoveServerEvents(context.Context, uuid.UUID, []uuid.UUID) error

	AddServerModerators(context.Context, uuid.UUID, model.Moderators) error
	RemoveServerModerators(context.Context, uuid.UUID, []uuid.UUID) error
}

func NewAPI(
	logger *zap.Logger,
	ctrl IController,
	sessionMiddleware *ihttp.SessionMiddleware,
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

			router.Method(http.MethodPost, "/server/tags", AddServerTags{API: api})
			router.Method(http.MethodDelete, "/server/tags", RemoveServerTags{API: api})

			router.Method(http.MethodPost, "/server/events", AddServerEvents{API: api})
			router.Method(http.MethodDelete, "/server/events", RemoveServerEvents{API: api})

			router.Method(http.MethodPost, "/server/moderators", AddServerModerators{API: api})
			router.Method(http.MethodDelete, "/server/moderators", RemoveServerModerators{API: api})

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
	})

	return &api
}

type API struct {
	Mux *chi.Mux

	logger *zap.Logger
	valid  *validatorv10.Validate
	ctrl   IController
}
