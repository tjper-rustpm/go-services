package rest

import (
	"encoding/json"
	"net/http"

	"github.com/tjper/rustcron/cmd/payment/controller"
	ihttp "github.com/tjper/rustcron/internal/http"
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

	url, err := ep.ctrl.BillingPortalSession(
		r.Context(),
		controller.BillingPortalSessionInput{
			ReturnURL:  b.ReturnURL,
			CustomerID: "",
		},
	)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	http.Redirect(w, r, url, http.StatusSeeOther)
}
