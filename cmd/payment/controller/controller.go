package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/internal/event"
	iuuid "github.com/tjper/rustcron/internal/uuid"

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

type IStream interface {
	Write(context.Context, []byte) error
}

func New(
	logger *zap.Logger,
	gorm *gorm.DB,
	staging *staging.Client,
	stripe IStripe,
	stream IStream,
) *Controller {
	return &Controller{
		logger:  logger,
		gorm:    gorm,
		staging: staging,
		stripe:  stripe,
		stream:  stream,
	}
}

type Controller struct {
	logger  *zap.Logger
	staging *staging.Client
	gorm    *gorm.DB
	stripe  IStripe
	stream  IStream
}

type CheckoutSessionInput struct {
	ServerID   uuid.UUID
	UserID     uuid.UUID
	CancelURL  string
	SuccessURL string
	CustomerID string
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

	var customerID *string
	if input.CustomerID != "" {
		customerID = &input.CustomerID
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
		Customer:          customerID,
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

	var subscription model.Subscription
	if err := ctrl.gorm.Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)

		sub := model.Subscription{
			StripeEventID: stripeEvent.ID,
		}
		exists, err := sub.ExistsWithStripeEventID(ctx, tx)
		if err != nil {
			return err
		}
		if exists {
			return ErrEventAlreadyProcessed
		}

		subscription = model.Subscription{
			StripeCheckoutID:     checkout.ID,
			StripeCustomerID:     checkout.Customer.ID,
			StripeSubscriptionID: checkout.Subscription.ID,
			StripeEventID:        stripeEvent.ID,
		}
		return tx.Create(&subscription).Error
	}); err != nil {
		return fmt.Errorf(
			"create subscription; eventID: %s, error: %w",
			stripeEvent.ID,
			err,
		)
	}

	if err := ctrl.customerCreated(ctx, stagedCheckout.UserID, checkout.Customer.ID); err != nil {
		return err
	}

	return ctrl.subscriptionCreated(
		ctx,
		subscription.ID,
		stagedCheckout.UserID,
		stagedCheckout.ServerID,
	)
}

func (ctrl Controller) ProcessInvoice(
	ctx context.Context,
	stripeEvent stripe.Event,
) error {
	var invoiceEvent stripe.Invoice
	if err := json.Unmarshal(stripeEvent.Data.Raw, &invoiceEvent); err != nil {
		return fmt.Errorf("unmarshal invoice; error: %w", err)
	}

	var invoice model.Invoice
	if err := ctrl.gorm.Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)

		invoice = model.Invoice{
			StripeEventID: stripeEvent.ID,
		}
		exists, err := invoice.ExistsWithStripeEventID(ctx, tx)
		if err != nil {
			return err
		}
		if exists {
			return ErrEventAlreadyProcessed
		}

		subscription := model.Subscription{}
		if res := tx.Where(
			"stripe_subscription_id = ?",
			invoiceEvent.Subscription.ID,
		).First(&subscription); res.Error != nil {
			return fmt.Errorf(
				"fetch invoice subscription; id: %s, error: %w",
				invoiceEvent.Subscription.ID,
				res.Error,
			)
		}

		invoice = model.Invoice{
			SubscriptionID: subscription.ID,
			Status:         model.InvoiceStatus(string(invoiceEvent.Status)),
			StripeEventID:  stripeEvent.ID,
		}
		return tx.Create(&invoice).Error
	}); err != nil {
		return fmt.Errorf(
			"create invoice; eventID: %s, error: %w",
			stripeEvent.ID,
			err,
		)
	}

	if invoice.Status == model.InvoiceStatusPaymentFailed {
		if err := ctrl.subscriptionDeleted(ctx, invoice.SubscriptionID); err != nil {
			return err
		}
	}
	return nil
}

func (ctrl Controller) UserSubscriptions(
	ctx context.Context,
	subscriptionIDs []uuid.UUID,
) ([]model.Subscription, error) {
	subscriptions := make([]model.Subscription, 0)
	if res := ctrl.gorm.
		Preload("Invoices").
		Where("id IN ?", iuuid.Strings(subscriptionIDs)).
		Find(&subscriptions); res.Error != nil {
		return nil, res.Error
	}

	return subscriptions, nil
}

func (ctrl Controller) customerCreated(
	ctx context.Context,
	userID uuid.UUID,
	customerID string,
) error {
	event := event.NewCustomerCreatedEvent(userID, customerID)

	b, err := json.Marshal(&event)
	if err != nil {
		return fmt.Errorf(
			"marshal customer created event; event-id: %s, error: %w",
			event.ID.String(),
			err,
		)
	}

	if err := ctrl.stream.Write(ctx, b); err != nil {
		return fmt.Errorf("stream write; event-id: %s, error: %w", event.ID.String(), err)
	}

	return nil
}

func (ctrl Controller) subscriptionCreated(
	ctx context.Context,
	subscriptionID uuid.UUID,
	userID uuid.UUID,
	serverID uuid.UUID,
) error {
	event := event.NewSubscriptionCreatedEvent(subscriptionID, userID, serverID)

	b, err := json.Marshal(&event)
	if err != nil {
		return fmt.Errorf(
			"marshal subscription created event; event-id: %s, error: %w",
			event.ID.String(),
			err,
		)
	}

	if err := ctrl.stream.Write(ctx, b); err != nil {
		return fmt.Errorf("stream subscription created write; event-id: %s, error: %w", event.ID.String(), err)
	}

	return nil
}

func (ctrl Controller) subscriptionDeleted(ctx context.Context, id uuid.UUID) error {
	event := event.NewSubscriptionDeleteEvent(id)

	b, err := json.Marshal(&event)
	if err != nil {
		return fmt.Errorf(
			"marshal subscription deleted event; event-id: %s, error: %w",
			event.ID.String(),
			err,
		)
	}

	if err := ctrl.stream.Write(ctx, b); err != nil {
		return fmt.Errorf("stream subscription deleted write; event-id: %s, error: %w", event.ID.String(), err)
	}

	return nil
}
