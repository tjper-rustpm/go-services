package rest

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/tjper/rustcron/cmd/payment/controller"
	ihttp "github.com/tjper/rustcron/internal/http"

	"github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
)

type EventConstructor interface {
	ConstructEvent([]byte, string) (stripe.Event, error)
}

type Stripe struct {
	API

	constructor EventConstructor
}

func (ep Stripe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	event, err := ep.constructor.ConstructEvent(
		b,
		r.Header.Get("Stripe-Signature"),
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
		fallthrough
	case "invoice.payment_failed":
		handler = ep.ctrl.ProcessInvoice
	default:
		ep.logger.Error("unknown webhook event", zap.String("type", event.Type))
		return
	}

	w.WriteHeader(http.StatusOK)

	err = handler(r.Context(), event)
	if errors.Is(err, controller.ErrEventAlreadyProcessed) {
		return
	}
	if err != nil {
		ep.logger.Error("stripe event handling", zap.Error(err))
	}
}
