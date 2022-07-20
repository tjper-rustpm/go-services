package rest

import (
	"context"
	"net/http"

	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/internal/gorm"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
	"github.com/tjper/rustcron/internal/validator"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	validatorv10 "github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
)

// IStore encompasses all interactions with the payment store.
type IStore interface {
	Create(context.Context, gorm.Creator) error
	First(context.Context, gorm.Firster) error

	FindByUserID(context.Context, gorm.FinderByUserID, uuid.UUID) error
	FindActiveSubscriptions(context.Context, *model.Servers) error
	IsSubscribedToServer(context.Context, *model.Customer, uuid.UUID) (bool, error)
}

// IStream encompasses all interactions with the event stream.
type IStream interface {
	Write(context.Context, []byte) error
}

// IStream encompasses all interactions with Stripe.
type IStripe interface {
	CheckoutSession(*stripe.CheckoutSessionParams) (string, error)
	BillingPortalSession(*stripe.BillingPortalSessionParams) (string, error)
	ConstructEvent([]byte, string) (stripe.Event, error)
}

// NewAPI creates a API instance.
func NewAPI(
	logger *zap.Logger,
	store IStore,
	staging *staging.Client,
	stream IStream,
	stripe IStripe,
	sessionMiddleware *ihttp.SessionMiddleware,
	healthz http.Handler,
) *API {
	api := API{
		Mux:     chi.NewRouter(),
		logger:  logger,
		valid:   validator.New(),
		staging: staging,
		store:   store,
		stripe:  stripe,
		stream:  stream,
	}

	api.Mux.Handle("/healthz", healthz)

	api.Mux.Use(
		sessionMiddleware.InjectSessionIntoCtx(),
		sessionMiddleware.Touch(),
		middleware.RequestLogger(ihttp.NewZapLogFormatter(logger)),
	)

	api.Mux.Route("/v1", func(router chi.Router) {
		router.Method(http.MethodPost, "/stripe", Stripe{API: api})
		router.Method(http.MethodGet, "/servers", Servers{API: api})

		router.Group(func(router chi.Router) {
			router.Use(sessionMiddleware.IsAuthenticated())

			router.Method(http.MethodPost, "/checkout", Checkout{API: api})
			router.Method(http.MethodPost, "/billing", Billing{API: api})
			router.Method(http.MethodGet, "/subscriptions", SubscriptionsEndpoint{API: api})

			router.Group(func(router chi.Router) {
				router.Use(sessionMiddleware.HasRole(session.RoleAdmin))

				router.Method(http.MethodPost, "/server", CreateServer{API: api})
			})
		})
	})

	return &api
}

// API is responsible for a REST API that manages access to payment related
// resources.
type API struct {
	Mux *chi.Mux

	logger *zap.Logger
	valid  *validatorv10.Validate

	staging *staging.Client
	store   IStore
	stripe  IStripe
	stream  IStream
}
