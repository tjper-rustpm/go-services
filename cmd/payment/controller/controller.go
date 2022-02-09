package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/cmd/payment/staging"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ErrEventAlreadyProcessed = errors.New("event already processed")

type IStripe interface {
	CheckoutSession(*stripe.CheckoutSessionParams) (string, error)
	BillingPortalSession(*stripe.BillingPortalSessionParams) (string, error)
}

func New(
	logger *zap.Logger,
	gorm *gorm.DB,
	staging *staging.Client,
	stripe IStripe,
) *Controller {
	return &Controller{
		logger:  logger,
		gorm:    gorm,
		staging: staging,
		stripe:  stripe,
	}
}

type Controller struct {
	logger  *zap.Logger
	staging *staging.Client
	gorm    *gorm.DB
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
	stripeEvent stripe.Event,
) error {
	var checkout stripe.CheckoutSession
	if err := json.Unmarshal(stripeEvent.Data.Raw, &checkout); err != nil {
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

	if err := ctrl.gorm.Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)

		event := model.Event{
			StripeEventID: stripeEvent.ID,
		}
		exists, err := event.Exists(ctx, tx)
		if err != nil {
			return err
		}
		if exists {
			return ErrEventAlreadyProcessed
		}

		subscription := &model.Subscription{
			ServerID:             stagedCheckout.ServerID,
			UserID:               stagedCheckout.UserID,
			StripeCheckoutID:     checkout.ID,
			StripeCustomerID:     checkout.Customer.ID,
			StripeSubscriptionID: checkout.Subscription.ID,
			Event:                event,
		}
		return tx.Create(subscription).Error
	}); err != nil {
		return fmt.Errorf(
			"create subscription; eventID: %s, error: %w",
			stripeEvent.ID,
			err,
		)
	}

	return nil
}

func (ctrl Controller) ProcessInvoice(
	ctx context.Context,
	stripeEvent stripe.Event,
) error {
	var invoiceEvent stripe.Invoice
	if err := json.Unmarshal(stripeEvent.Data.Raw, &invoiceEvent); err != nil {
		return fmt.Errorf("unmarshal invoice; error: %w", err)
	}

	if err := ctrl.gorm.Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)

		event := model.Event{
			StripeEventID: stripeEvent.ID,
		}
		exists, err := event.Exists(ctx, tx)
		if err != nil {
			return err
		}
		if exists {
			return ErrEventAlreadyProcessed
		}

		invoice := &model.Invoice{
			StripeSubscriptionID: invoiceEvent.Subscription.ID,
			Status:               model.InvoiceStatus(string(invoiceEvent.Status)),
			Event:                event,
		}
		return tx.Create(invoice).Error
	}); err != nil {
		return fmt.Errorf(
			"create invoice; eventID: %s, error: %w",
			stripeEvent.ID,
			err,
		)
	}

	return nil
}

func (ctrl Controller) UserSubscriptions(
	ctx context.Context,
	userID uuid.UUID,
) ([]model.Subscription, error) {
	subscriptions := make([]model.Subscription, 0)
	if res := ctrl.gorm.
		Preload("Invoices").
		Where("user_id = ?", userID).
		Find(&subscriptions); res.Error != nil {
		return nil, res.Error
	}

	return subscriptions, nil
}
