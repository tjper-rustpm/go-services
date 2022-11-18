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
	istripe "github.com/tjper/rustcron/internal/stripe"

	"github.com/google/uuid"
)

type Checkout struct{ API }

func (ep Checkout) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ServerID   uuid.UUID `json:"serverId" validate:"required"`
		SteamID    string    `json:"steamId" validate:"required"`
		CancelURL  string    `json:"cancelUrl" validate:"required,url"`
		SuccessURL string    `json:"successUrl" validate:"required,url"`
		PriceID    string    `json:"priceId" validate:"required,oneof=price_1LyigBCEcXRU8XL2L6eMGz6Y"`
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

	if !ep.checkoutEnabled {
		ihttp.ErrNotFound(w)
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

	url, err := ep.checkout(
		r.Context(),
		b.ServerID,
		b.SteamID,
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

func (ep Checkout) checkout(
	ctx context.Context,
	serverID uuid.UUID,
	steamID string,
	priceID string,
	cancelURL string,
	successURL string,
) (string, error) {
	expiresAt := time.Now().Add(time.Hour)

	clientReferenceID, err := ep.staging.StageCheckout(
		ctx,
		&staging.Checkout{
			ServerID: serverID,
			SteamID:  steamID,
		},
		expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("stage checkout session; error: %w", err)
	}

	emptyStripeCustomerID := ""

	checkout, err := istripe.NewCheckout(
		priceID,
		cancelURL,
		successURL,
		emptyStripeCustomerID,
		clientReferenceID,
		expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("while checking out: %w", err)
	}

	return ep.stripe.CheckoutSession(checkout)
}
