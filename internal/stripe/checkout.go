package stripe

import (
	"errors"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v72"
	"github.com/tjper/rustcron/cmd/payment/config"
)

var (
	// monthlyVipSubscriptionPriceID holds the monthly VIP price ID and should
	// be accessed via the MonthlyVipPriceID function.
	monthlyVipPriceID string

	// fiveDayVipPriceID holds the five day VIP price ID and should be accessed
	// via the FiveDayVipPriceID function.
	fiveDayVipPriceID string
)

// MonthlyVipPriceID uniquely identifies the price of a monthy VIP item.
// CAUTION: This function pulls the monthly VIP price ID from the config
// package an may result in an env-var read.
func MonthlyVipPriceID() string {
	if monthlyVipPriceID == "" {
		cfg := config.Load()
		monthlyVipPriceID = cfg.StripeMonthlyVIPPriceID()
	}
	return monthlyVipPriceID
}

// FiveDayVipPriceID uniquely identifies the price of a five day VIP item.
// CAUTION: This function pulls the five day VIP price ID from the config
// package an may result in an env-var read.
func FiveDayVipPriceID() string {
	if fiveDayVipPriceID == "" {
		cfg := config.Load()
		fiveDayVipPriceID = cfg.StripeFiveDayVIPPriceID()
	}
	return fiveDayVipPriceID
}

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
	case MonthlyVipPriceID():
		mode = stripe.CheckoutSessionModeSubscription
	case FiveDayVipPriceID():
		mode = stripe.CheckoutSessionModePayment
	default:
		return nil, fmt.Errorf("while building new checkout: %w", errUnrecognizedPriceID)
	}

	params := &stripe.CheckoutSessionParams{
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
	}

	if customerID != "" {
		params.Customer = stripe.String(customerID)
	}

	return params, nil
}
