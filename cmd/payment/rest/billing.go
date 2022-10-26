package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tjper/rustcron/internal/gorm"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"

	"github.com/stripe/stripe-go/v72"
)

type Billing struct{ API }

func (ep Billing) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ReturnURL string `json:"returnUrl" validate:"required,url"`
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

	sess, ok := session.FromContext(r.Context())
	if !ok {
		ihttp.ErrUnauthorized(w)
		return
	}

	customer, err := ep.store.FirstCustomerByUserID(r.Context(), sess.User.ID)
	if errors.Is(err, gorm.ErrNotFound) {
		ihttp.ErrNotFound(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	url, err := ep.stripe.BillingPortalSession(
		&stripe.BillingPortalSessionParams{
			ReturnURL: stripe.String(b.ReturnURL),
			Customer:  stripe.String(customer.StripeCustomerID),
		},
	)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)

	resp := &Redirect{
		URL: url,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
	}
}
