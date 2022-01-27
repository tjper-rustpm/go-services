package rest

import (
	"context"
	"io/ioutil"
	"net/http"

	ihttp "github.com/tjper/rustcron/internal/http"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/webhook"
	"go.uber.org/zap"
)

type Stripe struct {
	API

	stripeWebhookSecret string
}

func (ep Stripe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	event, err := webhook.ConstructEvent(
		b,
		r.Header.Get("Stripe-Signature"),
		ep.stripeWebhookSecret,
	)
	if err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	var handler func(context.Context, stripe.Event) error
	switch event.Type {
	case "checkout.session.completed":
		handler = ep.ctrl.CheckoutSessionComplete
	case "invoice.paid":
	case "invoice.payment_failed":
		handler = ep.ctrl.ProcessInvoice
	default:
		ep.logger.Error("unknown webhook event", zap.String("type", event.Type))
		return
	}

	if err := handler(r.Context(), event); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}
}
