// Package stream provides and API for launching a Handler that reads and
// processes all payments related events from the underlying stream.
package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/payment/db"
	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/internal/event"
	"github.com/tjper/rustcron/internal/gorm"
	imodel "github.com/tjper/rustcron/internal/model"
	"github.com/tjper/rustcron/internal/stream"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
)

// IStore encompasses all interactions with the payment store.
type IStore interface {
	First(context.Context, gorm.Firster) error
	FirstByStripeEventID(context.Context, db.FirsterByStripeEventID) error

	CreateSubscription(context.Context, *model.Subscription, *model.Customer, uuid.UUID) error
	CreateInvoice(context.Context, *model.Invoice, string) error
}

// IStream encompasses all interactions with the event stream.
type IStream interface {
	Claim(context.Context, time.Duration) (*stream.Message, error)
	Read(context.Context) (*stream.Message, error)
	Write(context.Context, []byte) error
}

// NewHandler creates a Handler instance.
func NewHandler(
	logger *zap.Logger,
	staging *staging.Client,
	store IStore,
	stream IStream,
) *Handler {
	return &Handler{
		logger:  logger,
		staging: staging,
		store:   store,
		stream:  stream,
	}
}

// Handler is responsible for reading and processing payment related events
// from the underlying IStream passed into NewHandler.
type Handler struct {
	logger  *zap.Logger
	staging *staging.Client
	store   IStore
	stream  IStream
}

// Launch reads and processes the underlying IStream. This is a blocking
// function. The context may be cancelled to shutdown the handler.
func (h Handler) Launch(ctx context.Context) error {
	for {
		m, err := h.read(ctx)
		if err != nil {
			return fmt.Errorf("stream Handler.read: %w", err)
		}

		eventI, err := event.Parse(m.Payload)
		if err != nil {
			h.logger.Error("parse event hash", zap.Error(err))
			continue
		}

		switch e := eventI.(type) {
		case *event.StripeWebhookEvent:
			err = h.handleStripeEvent(ctx, e)
		default:
			h.logger.Sugar().Debugf("unrecognized event; type: %T", e)
		}
		if err != nil {
			h.logger.Error("handle stream event", zap.Error(err))
			continue
		}

		if err := m.Ack(ctx); err != nil {
			h.logger.Error("acknowledge stream event", zap.Error(err))
		}
	}
}

func (h Handler) handleStripeEvent(ctx context.Context, event *event.StripeWebhookEvent) error {
	stripeEvent := event.StripeEvent

	var handler func(context.Context, stripe.Event) error
	switch stripeEvent.Type {
	case "checkout.session.completed":
		handler = h.processCheckoutSessionComplete
	case "invoice.paid":
		fallthrough
	case "invoice.payment_failed":
		handler = h.processInvoice
	default:
		h.logger.Warn("unknown stripe webhook event", zap.String("type", stripeEvent.Type))
		return nil
	}

	if err := handler(ctx, stripeEvent); err != nil {
		h.logger.Error("stripe event handling", zap.Error(err))
		return err
	}

	return nil
}

func (h Handler) processCheckoutSessionComplete(ctx context.Context, event stripe.Event) error {
	var checkout stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &checkout); err != nil {
		return fmt.Errorf("unmarshal checkout; error: %w", err)
	}

	var errstr string
	switch {
	case event.ID == "":
		errstr = "event ID empty"
	case checkout.ClientReferenceID == "":
		errstr = "checkout ClientReferenceID empty"
	case checkout.ID == "":
		errstr = "checkout ID empty"
	case checkout.Subscription == nil:
		errstr = "checkout Subscription nil"
	case checkout.Subscription.ID == "":
		errstr = "checkout Subscription ID empty"
	case checkout.Customer == nil:
		errstr = "checkout Customer nil"
	case checkout.Customer.ID == "":
		errstr = "checkout Customer ID empty"
	}
	if errstr != "" {
		h.logger.Warn(errstr)
		return nil
	}

	stagedCheckout, err := h.staging.FetchCheckout(ctx, checkout.ClientReferenceID)
	if err != nil {
		return fmt.Errorf(
			"fetch staged checkout; id: %s, error: %w",
			checkout.ClientReferenceID,
			err,
		)
	}

	subscription := &model.Subscription{
		StripeCheckoutID:     checkout.ID,
		StripeSubscriptionID: checkout.Subscription.ID,
		StripeEventID:        event.ID,
	}

	err = h.store.FirstByStripeEventID(ctx, subscription)
	if err == nil {
		// Subscription has already been processed, return early.
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrNotFound) {
		return fmt.Errorf("find subscription be event ID; error: %w", err)
	}

	if err := h.store.CreateSubscription(
		ctx,
		subscription,
		&model.Customer{
			UserID:           stagedCheckout.UserID,
			StripeCustomerID: checkout.Customer.ID,
			SteamID:          stagedCheckout.SteamID,
		},
		stagedCheckout.ServerID,
	); err != nil {
		return fmt.Errorf(
			"create subscription; eventID: %s, error: %w",
			event.ID,
			err,
		)
	}

	return nil
}

var errInvoiceSubscriptionDNE = errors.New("invoice subscription does not exist")

func (h Handler) processInvoice(ctx context.Context, event stripe.Event) error {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	var errstr string
	switch {
	case event.ID == "":
		errstr = "event ID empty"
	case invoice.Status == "":
		errstr = "invoice Status empty"
	case invoice.Subscription == nil:
		errstr = "invoice Subscription nil"
	case invoice.Subscription.ID == "":
		errstr = "invoice Subscription ID empty"
	}
	if errstr != "" {
		h.logger.Warn(errstr)
		return nil
	}

	invoiceModel := &model.Invoice{
		Status:        model.InvoiceStatus(string(invoice.Status)),
		StripeEventID: event.ID,
	}

	err := h.store.FirstByStripeEventID(ctx, invoiceModel)
	if err == nil {
		// Invoice has already been processed, return early.
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrNotFound) {
		return fmt.Errorf("store.FindByStripeEventID: %w", err)
	}

	err = h.store.CreateInvoice(ctx, invoiceModel, invoice.Subscription.ID)
	if errors.Is(err, gorm.ErrNotFound) {
		return errInvoiceSubscriptionDNE
	}
	if err != nil {
		return fmt.Errorf("store.CreateInvoice: %w", err)
	}

	subscription := &model.Subscription{
		Model: imodel.Model{ID: invoiceModel.SubscriptionID},
	}
	err = h.store.First(ctx, subscription)
	if err != nil {
		return fmt.Errorf("store.First: %w", err)
	}

	if invoice.Status == stripe.InvoiceStatusPaid {
		if err = h.invoicePaid(
			ctx,
			subscription.ID,
			subscription.Server.ID,
			subscription.Customer.SteamID,
		); err != nil {
			return fmt.Errorf("Handler.invoicePaid: %w", err)
		}
		return nil
	}

	if err = h.invoicePaymentFailure(ctx, subscription.ID); err != nil {
		return fmt.Errorf("Handler.invoicePaymentFailure: %w", err)
	}
	return nil
}

func (h Handler) invoicePaid(
	ctx context.Context,
	subsriptionID uuid.UUID,
	serverID uuid.UUID,
	steamID string,
) error {
	paid := event.NewInvoicePaidEvent(subsriptionID, serverID, steamID)

	b, err := json.Marshal(&paid)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	if err := h.stream.Write(ctx, b); err != nil {
		return fmt.Errorf("stream.Write: %w", err)
	}
	return nil
}

func (h Handler) invoicePaymentFailure(
	ctx context.Context,
	subscriptionID uuid.UUID,
) error {
	paymentFailure := event.NewInvoicePaymentFailure(subscriptionID)

	b, err := json.Marshal(&paymentFailure)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	if err := h.stream.Write(ctx, b); err != nil {
		return fmt.Errorf("stream.Write: %w", err)
	}

	return nil
}

func (h Handler) read(ctx context.Context) (*stream.Message, error) {
	m, err := h.stream.Claim(ctx, time.Minute)
	if err == nil {
		return m, nil
	}
	if err != nil && !errors.Is(err, stream.ErrNoPending) {
		return nil, fmt.Errorf("stream.Claim: %w", err)
	}

	// stream.Claim has returned stream.ErrNoPending, therefore we may read
	// the stream.
	m, err = h.stream.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("stream.Read: %w", err)
	}
	return m, nil
}
