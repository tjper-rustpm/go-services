package rest

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/tjper/rustcron/internal/event"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type Stripe struct {
	API
}

func (ep Stripe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	stripeEvent, err := ep.stripe.ConstructEvent(
		b,
		r.Header.Get("Stripe-Signature"),
	)
	if err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	stripeWebhookEvent := event.NewStripeWebhookEvent(stripeEvent)

	b, err = json.Marshal(&stripeWebhookEvent)
	if err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	if err := ep.stream.Write(r.Context(), b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
	}

	w.WriteHeader(http.StatusOK)
}
