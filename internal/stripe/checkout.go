package stripe

import (
	"errors"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v72"
)

// Price is a Stripe priced item. This a unique set of characters provided by
// Stripe that is used to identify and utilize Stripe pricing.
type Price string

const (
	// MonthlyVipSubscription uniquely identifies the price of a monthy VIP
	// subscription item.
	MonthlyVipSubscription Price = "price_1KLJWjCEcXRU8XL2TVKcLGUO"
	// WeeklyVipOneTime uniquely identifies the price of a one-time weekly VIP
	// item.
	WeeklyVipOneTime Price = "price_1LyigBCEcXRU8XL2L6eMGz6Y"
)

var errUnrecognizedPriceID = errors.New("unrecognized price ID")

func NewCheckout(
	priceID string,
	cancelURL string,
	successURL string,
	customerID string,
	clientReferenceID string,
	expiresAt time.Time,
) (*stripe.CheckoutSessionParams, error) {
	var mode stripe.CheckoutSessionMode
	switch priceID {
	case string(MonthlyVipSubscription):
		mode = stripe.CheckoutSessionModeSubscription
	case string(WeeklyVipOneTime):
		mode = stripe.CheckoutSessionModePayment
	default:
		return nil, fmt.Errorf("while building new checkout: %w", errUnrecognizedPriceID)
	}

	return &stripe.CheckoutSessionParams{
		CancelURL:  stripe.String(cancelURL),
		SuccessURL: stripe.String(successURL),
		Mode:       stripe.String(string(mode)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		ClientReferenceID: stripe.String(clientReferenceID),
		ExpiresAt:         stripe.Int64(expiresAt.Unix()),
		Customer:          stripe.String(customerID),
	}, nil
}
