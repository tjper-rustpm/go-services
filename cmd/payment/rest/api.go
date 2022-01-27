package rest

import (
	"context"
	"net/http"

	"github.com/tjper/rustcron/cmd/payment/controller"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/validator"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	validatorv10 "github.com/go-playground/validator/v10"
	"github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
)

type IController interface {
	CheckoutSession(context.Context, controller.CheckoutSessionInput) (string, error)
	BillingPortalSession(context.Context, controller.BillingPortalSessionInput) (string, error)
	CheckoutSessionComplete(context.Context, stripe.Event) error
	ProcessInvoice(context.Context, stripe.Event) error
}

func NewAPI(
	logger *zap.Logger,
	ctrl IController,
	sessionMiddleware *ihttp.SessionMiddleware,
	stripeWebhookSecret string,
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
		router.Method(
			http.MethodPost,
			"/payment/stripe",
			Stripe{API: api, stripeWebhookSecret: stripeWebhookSecret},
		)

		router.Group(func(router chi.Router) {
			api.Mux.Use(sessionMiddleware.IsAuthenticated())

			router.Method(http.MethodPost, "/payment/checkout", Checkout{API: api})
			router.Method(http.MethodPost, "/payment/billing", Billing{API: api})
		})
	})

	return &api
}

type API struct {
	Mux *chi.Mux

	logger *zap.Logger
	valid  *validatorv10.Validate
	ctrl   IController
}
