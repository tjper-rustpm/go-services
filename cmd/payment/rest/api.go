package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/payment/model"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
	"github.com/tjper/rustcron/internal/validator"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	validatorv10 "github.com/go-playground/validator/v10"
	"github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
)

// IStore encompasses all interactions with the payment store.
type IStore interface {
	FirstServerByID(context.Context, uuid.UUID) (*model.Server, error)
	FirstCustomerByUserID(context.Context, uuid.UUID) (*model.Customer, error)
	FindVipsByUserID(context.Context, uuid.UUID) (model.Vips, error)
	FindServers(context.Context) (model.Servers, error)
	CreateServer(context.Context, *model.Server) error
	UpdateServer(context.Context, uuid.UUID, map[string]interface{}) (*model.Server, error)
	IsServerVipBySteamID(context.Context, uuid.UUID, string) (bool, error)
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

// IStaging encompasses all interactions with checkout staging.
type IStaging interface {
	StageCheckout(context.Context, interface{}, time.Time) (string, error)
	FetchCheckout(context.Context, string) (interface{}, error)
}

// ISessionMiddleware encompasses all interactions with session related
// middleware.
type ISessionMiddleware interface {
	InjectSessionIntoCtx() func(http.Handler) http.Handler
	Touch() func(http.Handler) http.Handler
	HasRole(session.Role) func(http.Handler) http.Handler
	IsAuthenticated() func(http.Handler) http.Handler
}

// NewAPI creates a API instance.
func NewAPI(
	logger *zap.Logger,
	store IStore,
	staging IStaging,
	stream IStream,
	stripe IStripe,
	sessionMiddleware ISessionMiddleware,
	healthz http.Handler,
	options ...Option,
) *API {
	api := API{
		Mux:             chi.NewRouter(),
		logger:          logger,
		valid:           validator.New(),
		staging:         staging,
		store:           store,
		stripe:          stripe,
		stream:          stream,
		checkoutEnabled: true,
	}

	for _, option := range options {
		option(&api)
	}

	api.Mux.Handle("/healthz", healthz)

	api.Mux.Route("/v1", func(router chi.Router) {
		router.Use(
			sessionMiddleware.InjectSessionIntoCtx(),
			sessionMiddleware.Touch(),
			middleware.RequestLogger(ihttp.NewZapLogFormatter(logger)),
		)

		router.Method(http.MethodPost, "/stripe", Stripe{API: api})
		router.Method(http.MethodGet, "/servers", Servers{API: api})
    router.Method(http.MethodPost, "/checkout", Checkout{API: api})

		router.Group(func(router chi.Router) {
			router.Use(sessionMiddleware.IsAuthenticated())

			router.Method(http.MethodPost, "/billing", Billing{API: api})
			router.Method(http.MethodPost, "/subscription/checkout", SubscriptionCheckout{API: api})
			router.Method(http.MethodGet, "/subscriptions", SubscriptionsEndpoint{API: api})

			router.Group(func(router chi.Router) {
				router.Use(sessionMiddleware.HasRole(session.RoleAdmin))

				router.Method(http.MethodPost, "/server", CreateServer{API: api})
				router.Method(http.MethodPatch, "/server", UpdateServer{API: api})
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

	staging IStaging
	store   IStore
	stripe  IStripe
	stream  IStream

	checkoutEnabled bool
}

// Option configures the API instance. Option instances are typically used
// with NewAPI to configure an API instance.
type Option func(*API)

// WithCheckout creates an Option that enables/disables payment checkout.
func WithCheckout(enabled bool) Option {
	return func(api *API) {
		api.checkoutEnabled = enabled
	}
}
