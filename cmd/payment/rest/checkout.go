package rest

import (
	"encoding/json"
	"net/http"

	"github.com/tjper/rustcron/cmd/payment/controller"
	ihttp "github.com/tjper/rustcron/internal/http"

	"github.com/google/uuid"
)

type Checkout struct{ API }

func (ep Checkout) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ServerID   uuid.UUID `json:"serverId" validate:"required"`
		UserID     uuid.UUID `json:"userId" validate:"required"`
		CancelURL  string    `json:"cancelUrl" validate:"required,url"`
		SuccessURL string    `json:"successUrl" validate:"required,url"`
		PriceID    string    `json:"priceId" validate:"required,oneof=prod_L1MFlCUj2bk2j0"`
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
			ServerID:   b.ServerID,
			UserID:     b.UserID,
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
