package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v72"
	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/internal/gorm"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"

	"github.com/google/uuid"
)

type Checkout struct{ API }

func (ep Checkout) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ServerID   uuid.UUID `json:"serverId" validate:"required"`
		CancelURL  string    `json:"cancelUrl" validate:"required,url"`
		SuccessURL string    `json:"successUrl" validate:"required,url"`
		PriceID    string    `json:"priceId" validate:"required,oneof=price_1KLJWjCEcXRU8XL2TVKcLGUO"`
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

	if !sess.User.IsSteamIDAssociated() {
		ihttp.ErrForbidden(w)
		return
	}

	customerID, err := ep.customerID(r.Context(), sess.User.ID)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	url, err := ep.checkout(
		r.Context(),
		b.ServerID,
		sess.User.ID,
		sess.User.SteamID,
		customerID,
		b.PriceID,
		b.CancelURL,
		b.SuccessURL,
	)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (ep Checkout) customerID(ctx context.Context, userID uuid.UUID) (string, error) {
	customer := &model.Customer{
		UserID: userID,
	}
	err := ep.store.First(ctx, customer)
	if errors.Is(err, gorm.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("fetch customer ID; error: %w", err)
	}

	return customer.StripeCustomerID, nil
}

func (ep Checkout) checkout(
	ctx context.Context,
	serverID uuid.UUID,
	userID uuid.UUID,
	steamID string,
	customerID string,
	priceID string,
	cancelURL string,
	successURL string,
) (string, error) {
	expiresAt := time.Now().Add(time.Hour)

	clientReferenceID, err := ep.staging.StageCheckout(
		ctx,
		staging.Checkout{
			ServerID: serverID,
			UserID:   userID,
			SteamID:  steamID,
		},
		expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("stage checkout session; error: %w", err)
	}

	var ptrCustomerID *string
	if customerID != "" {
		ptrCustomerID = &customerID
	}

	params := &stripe.CheckoutSessionParams{
		CancelURL:  stripe.String(cancelURL),
		SuccessURL: stripe.String(successURL),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		ClientReferenceID: stripe.String(clientReferenceID),
		ExpiresAt:         stripe.Int64(expiresAt.Unix()),
		Customer:          ptrCustomerID,
	}

	return ep.stripe.CheckoutSession(params)
}
