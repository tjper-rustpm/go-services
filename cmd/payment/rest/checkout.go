package rest

import (
	"encoding/json"
	"net/http"

	"github.com/tjper/rustcron/cmd/payment/controller"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type Checkout struct{ API }

func (ep Checkout) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		CancelURL  string `json:"cancelUrl" validate:"required,url"`
		SuccessURL string `json:"successUrl" validate:"required,url"`
		PriceID    string `json:"priceId" validate:"required"`
	}

	var b body
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	url, err := ep.ctrl.CheckoutSession(
		r.Context(),
		controller.CheckoutSessionInput{
			CancelURL:  b.CancelURL,
			SuccessURL: b.SuccessURL,
			PriceID:    b.PriceID,
		},
	)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	http.Redirect(w, r, url, http.StatusSeeOther)
}
