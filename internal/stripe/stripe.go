package stripe

import (
	"fmt"

	"github.com/stripe/stripe-go/v72"
	billing "github.com/stripe/stripe-go/v72/billingportal/session"
	checkout "github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/webhook"
)

func New(
	webhookSecret string,
	billing *billing.Client,
	checkout *checkout.Client,
) *Stripe {
	return &Stripe{
		webhookSecret: webhookSecret,
		billing:       billing,
		checkout:      checkout,
	}
}

type Stripe struct {
	webhookSecret string

	billing  *billing.Client
	checkout *checkout.Client
}

func (s Stripe) CheckoutSession(params *stripe.CheckoutSessionParams) (string, error) {
	sess, err := s.checkout.New(params)
	if err != nil {
		return "", fmt.Errorf("new checkout session; error: %w", err)
	}
	return sess.URL, nil
}

func (s Stripe) BillingPortalSession(params *stripe.BillingPortalSessionParams) (string, error) {
	sess, err := s.billing.New(params)
	if err != nil {
		return "", fmt.Errorf("new billing portal session; error: %w", err)
	}
	return sess.URL, nil
}

func (s Stripe) ConstructEvent(b []byte, signature string) (stripe.Event, error) {
	return webhook.ConstructEvent(b, signature, s.webhookSecret)
}
