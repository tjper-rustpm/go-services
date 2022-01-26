package controller

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v72"
	billing "github.com/stripe/stripe-go/v72/billingportal/session"
	checkout "github.com/stripe/stripe-go/v72/checkout/session"
	"go.uber.org/zap"
)

func New(
	logger *zap.Logger,
	checkout *checkout.Client,
	billing *billing.Client,
) *Controller {
	return &Controller{
		logger:   logger,
		checkout: checkout,
		billing:  billing,
	}
}

type Controller struct {
	logger   *zap.Logger
	checkout *checkout.Client
	billing  *billing.Client
}

type CheckoutSessionInput struct {
	CancelURL  string
	SuccessURL string
	PriceID    string
}

func (ctrl Controller) CheckoutSession(
	ctx context.Context,
	input CheckoutSessionInput,
) (string, error) {
	params := &stripe.CheckoutSessionParams{
		CancelURL:  stripe.String(input.CancelURL),
		SuccessURL: stripe.String(input.SuccessURL),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(input.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
	}

	sess, err := ctrl.checkout.New(params)
	if err != nil {
		return "", fmt.Errorf("new checkout session; price-id: %s, error: %w", input.PriceID, err)
	}

	return sess.URL, nil
}

type BillingPortalSessionInput struct {
	CustomerID string
	ReturnURL  string
}

func (ctrl Controller) BillingPortalSession(
	ctx context.Context,
	input BillingPortalSessionInput,
) (string, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(input.CustomerID),
		ReturnURL: stripe.String(input.ReturnURL),
	}
	sess, err := ctrl.billing.New(params)
	if err != nil {
		return "", fmt.Errorf(
			"new billing portal session; customer-id: %s, error: %w",
			input.CustomerID,
			err,
		)
	}

	return sess.URL, nil
}
