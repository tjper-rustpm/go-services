package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/payment/db"
	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/cmd/payment/staging"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
)

type IStripe interface {
	CheckoutSession(*stripe.CheckoutSessionParams) (string, error)
	BillingPortalSession(*stripe.BillingPortalSessionParams) (string, error)
}

func New(
	logger *zap.Logger,
	store *db.Store,
	staging *staging.Client,
	stripe IStripe,
) *Controller {
	return &Controller{
		logger:  logger,
		store:   store,
		staging: staging,
		stripe:  stripe,
	}
}

type Controller struct {
	logger  *zap.Logger
	staging *staging.Client
	store   *db.Store
	stripe  IStripe
}

type CheckoutSessionInput struct {
	ServerID   uuid.UUID
	UserID     uuid.UUID
	CancelURL  string
	SuccessURL string
	PriceID    string
}

func (ctrl Controller) CheckoutSession(
	ctx context.Context,
	input CheckoutSessionInput,
) (string, error) {
	expiresAt := time.Now().Add(time.Hour)

	clientReferenceID, err := ctrl.staging.StageCheckout(
		ctx,
		staging.Checkout{ServerID: input.ServerID, UserID: input.UserID},
		expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("stage checkout session; error: %w", err)
	}

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
		ClientReferenceID: stripe.String(clientReferenceID),
		ExpiresAt:         stripe.Int64(expiresAt.Unix()),
	}

	return ctrl.stripe.CheckoutSession(params)
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

	return ctrl.stripe.BillingPortalSession(params)
}

func (ctrl Controller) CheckoutSessionComplete(
	ctx context.Context,
	event stripe.Event,
) error {
	var checkout stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &checkout); err != nil {
		return fmt.Errorf("unmarshal checkout; error: %w", err)
	}

	stagedCheckout, err := ctrl.staging.FetchCheckout(ctx, checkout.ClientReferenceID)
	if err != nil {
		return fmt.Errorf(
			"fetch staged checkout; id: %s, error: %w",
			checkout.ClientReferenceID,
			err,
		)
	}

	subscription := &model.Subscription{
		ServerID:             stagedCheckout.ServerID,
		UserID:               stagedCheckout.UserID,
		StripeCheckoutID:     checkout.ID,
		StripeCustomerID:     checkout.Customer.ID,
		StripeSubscriptionID: checkout.Subscription.ID,
	}
	return ctrl.store.Create(ctx, subscription)
}

func (ctrl Controller) ProcessInvoice(
	ctx context.Context,
	event stripe.Event,
) error {
	var invoiceEvent stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoiceEvent); err != nil {
		return fmt.Errorf("unmarshal invoice; error: %w", err)
	}

	invoice := &model.Invoice{
		StripeSubscriptionID: invoiceEvent.Subscription.ID,
		Status:               model.InvoiceStatus(string(invoiceEvent.Status)),
	}
	return ctrl.store.Create(ctx, invoice)
}
