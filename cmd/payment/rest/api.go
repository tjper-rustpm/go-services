package rest

import (
	"context"
	"net/http"

	"github.com/tjper/rustcron/cmd/payment/controller"
	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/internal/gorm"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
	"github.com/tjper/rustcron/internal/validator"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	validatorv10 "github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	stripev72 "github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
)

type IStore interface {
	Create(context.Context, gorm.Creator) error
}

type IController interface {
	CheckoutSession(context.Context, controller.CheckoutSessionInput) (string, error)
	BillingPortalSession(context.Context, controller.BillingPortalSessionInput) (string, error)

	UserSubscriptions(context.Context, []uuid.UUID) ([]model.Subscription, error)

	CheckoutSessionComplete(context.Context, stripev72.Event) error
	ProcessInvoice(context.Context, stripev72.Event) error
}

func NewAPI(
	logger *zap.Logger,
	ctrl IController,
	sessionMiddleware *ihttp.SessionMiddleware,
	eventConstructor EventConstructor,
) *API {
	api := API{
		Mux:    chi.NewRouter(),
		logger: logger,
		valid:  validator.New(),
		ctrl:   ctrl,
	}

	api.Mux.Use(
		sessionMiddleware.InjectSessionIntoCtx(),
		sessionMiddleware.Touch(),
		middleware.RequestLogger(ihttp.NewZapLogFormatter(logger)),
	)

	api.Mux.Route("/v1", func(router chi.Router) {
		router.Method(http.MethodPost, "/stripe", Stripe{API: api, constructor: eventConstructor})

		router.Group(func(router chi.Router) {
			router.Use(sessionMiddleware.IsAuthenticated())

			router.Method(http.MethodPost, "/checkout", Checkout{API: api})
			router.Method(http.MethodPost, "/billing", Billing{API: api})
			router.Method(http.MethodGet, "/subscriptions", Subscriptions{API: api})

			router.Group(func(router chi.Router) {
				router.Use(sessionMiddleware.HasRole(session.RoleAdmin))

				router.Method(http.MethodPost, "/server-subscription-limits", ServerSubscriptionLimits{API: api})
			})
		})
	})

	return &api
}

type API struct {
	Mux *chi.Mux

	logger *zap.Logger
	valid  *validatorv10.Validate
	ctrl   IController
	store  IStore
}
