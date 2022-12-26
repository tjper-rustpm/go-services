package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/internal/gorm"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
	istripe "github.com/tjper/rustcron/internal/stripe"

	"github.com/google/uuid"
)

type SubscriptionCheckout struct{ API }

func (ep SubscriptionCheckout) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ServerID   uuid.UUID `json:"serverId" validate:"required"`
		SteamID    string    `json:"steamId" validate:"required"`
		CancelURL  string    `json:"cancelUrl" validate:"required,url"`
		SuccessURL string    `json:"successUrl" validate:"required,url"`
		PriceID    string    `json:"priceId" validate:"required"`
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

	tag := fmt.Sprintf("required,oneof=%s", istripe.MonthlyVipPriceID())
	if err := ep.valid.Var(b.PriceID, tag); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	if !ep.checkoutEnabled {
		ihttp.ErrNotFound(w)
		return
	}

	sess, ok := session.FromContext(r.Context())
	if !ok {
		ihttp.ErrUnauthorized(w)
		return
	}

	// Ensure ServerID used in checkout has an associated Server.
	_, err := ep.store.FirstServerByID(r.Context(), b.ServerID)
	if errors.Is(err, gorm.ErrNotFound) {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	// Ensure Steam ID passed is not already a VIP on the specified server.
	isVip, err := ep.store.IsServerVipBySteamID(r.Context(), b.ServerID, b.SteamID)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}
	if isVip {
		ihttp.ErrConflict(w)
		return
	}

	// getStripeCustomerID will be empty "", if user has not made a purchase before. The
	// ep.checkout method should be able to handle an empty customerID.
	stripeCustomerID, err := ep.getStripeCustomerIDByUserID(r.Context(), sess.User.ID)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	url, err := ep.checkout(
		r.Context(),
		b.ServerID,
		sess.User.ID,
		b.SteamID,
		stripeCustomerID,
		b.PriceID,
		b.CancelURL,
		b.SuccessURL,
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

func (ep SubscriptionCheckout) getStripeCustomerIDByUserID(ctx context.Context, userID uuid.UUID) (string, error) {
	customer, err := ep.store.FirstCustomerByUserID(ctx, userID)
	if errors.Is(err, gorm.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("while retrieving customer ID; error: %w", err)
	}

	return customer.StripeCustomerID, nil
}

func (ep SubscriptionCheckout) checkout(
	ctx context.Context,
	serverID uuid.UUID,
	userID uuid.UUID,
	steamID string,
	stripeCustomerID string,
	priceID string,
	cancelURL string,
	successURL string,
) (string, error) {
	expiresAt := time.Now().Add(time.Hour)

	clientReferenceID, err := ep.staging.StageCheckout(
		ctx,
		&staging.UserCheckout{
			Checkout: staging.Checkout{
				ServerID: serverID,
				SteamID:  steamID,
				PriceID:  priceID,
			},
			UserID: userID,
		},
		expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("stage checkout session; error: %w", err)
	}

	checkout, err := istripe.NewCheckout(
		priceID,
		cancelURL,
		successURL,
		stripeCustomerID,
		clientReferenceID,
		expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("while checking out: %w", err)
	}

	return ep.stripe.CheckoutSession(checkout)
}
