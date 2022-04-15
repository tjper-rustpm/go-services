package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tjper/rustcron/cmd/payment/controller"
	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/internal/gorm"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
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

	customer := &model.Customer{
		UserID: sess.User.ID,
	}
	err := ep.store.First(r.Context(), customer)
	if errors.Is(err, gorm.ErrNotFound) {
		ihttp.ErrNotFound(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	url, err := ep.ctrl.BillingPortalSession(
		r.Context(),
		controller.BillingPortalSessionInput{
			ReturnURL:  b.ReturnURL,
			CustomerID: customer.StripeCustomerID,
		},
	)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	http.Redirect(w, r, url, http.StatusSeeOther)
}
