package rest

import (
	"context"
	"net/http"

	"github.com/tjper/rustcron/cmd/payment/db"
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

type IStore interface {
	Create(context.Context, gorm.Creator) error
	First(context.Context, gorm.Firster) error

	CreateSubscription(context.Context, *model.Subscription, *model.Customer, uuid.UUID) error
	CreateInvoice(context.Context, *model.Invoice, string) error

	FirstByStripeEventID(context.Context, db.FirsterByStripeEventID) error
	FindByUserID(context.Context, gorm.FinderByUserID, uuid.UUID) error
	FindActiveSubscriptions(context.Context, *model.Servers) error
}

type IStream interface {
	Write(context.Context, []byte) error
}

type IStripe interface {
	CheckoutSession(*stripe.CheckoutSessionParams) (string, error)
	BillingPortalSession(*stripe.BillingPortalSessionParams) (string, error)
	ConstructEvent([]byte, string) (stripe.Event, error)
}

func NewAPI(
	logger *zap.Logger,
	store IStore,
	staging *staging.Client,
	stream IStream,
	stripe IStripe,
	sessionMiddleware *ihttp.SessionMiddleware,
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

	api.Mux.Use(
		sessionMiddleware.InjectSessionIntoCtx(),
		sessionMiddleware.Touch(),
		middleware.RequestLogger(ihttp.NewZapLogFormatter(logger)),
	)

	api.Mux.Route("/v1", func(router chi.Router) {
		router.Method(http.MethodPost, "/stripe", Stripe{API: api})
		router.Method(http.MethodGet, "/server", Server{API: api})

		router.Group(func(router chi.Router) {
			router.Use(sessionMiddleware.IsAuthenticated())

			router.Method(http.MethodPost, "/checkout", Checkout{API: api})
			router.Method(http.MethodPost, "/billing", Billing{API: api})
			router.Method(http.MethodGet, "/subscriptions", Subscriptions{API: api})

			router.Group(func(router chi.Router) {
				router.Use(sessionMiddleware.HasRole(session.RoleAdmin))

				router.Method(http.MethodPost, "/server", CreateServer{API: api})
			})
		})
	})

	return &api
}

type API struct {
	Mux *chi.Mux

	logger *zap.Logger
	valid  *validatorv10.Validate

	staging *staging.Client
	store   IStore
	stripe  IStripe
	stream  IStream
}
