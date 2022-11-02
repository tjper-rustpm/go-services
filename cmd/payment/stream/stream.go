// Package stream provides and API for launching a Handler that reads and
// processes all payments related events from the underlying stream.
package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/internal/event"
	"github.com/tjper/rustcron/internal/gorm"
	"github.com/tjper/rustcron/internal/stream"
	istripe "github.com/tjper/rustcron/internal/stripe"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
)

// IStore encompasses all interactions with the payment store.
type IStore interface {
	FirstSubscriptionByID(context.Context, uuid.UUID) (*model.Subscription, error)
	FirstVipByStripeEventID(context.Context, string) (*model.Subscription, error)
	FirstInvoiceByStripeEventID(context.Context, string) (*model.Invoice, error)
	CreateVip(context.Context, *model.Vip, *model.Customer) error
	CreateVipSubscription(context.Context, *model.Vip, *model.Subscription, *model.Customer) error
	AddInvoiceToVipSubscription(context.Context, string, *model.Invoice) (*model.Vip, error)
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

// handleStripeEvent passes the handling of an event to sub-handler.
// CAUTION: If an error is returned, the event will not be acknowledged and
// will be reprocessed at a later time.
func (h Handler) handleStripeEvent(ctx context.Context, event *event.StripeWebhookEvent) error {
	stripeEvent := event.StripeEvent
	if stripeEvent.ID == "" {
		h.logger.Warn("while handling stripe event: stripe event ID empty; discarding event")
		return nil
	}

	var err error
	switch stripeEvent.Type {
	case "checkout.session.completed":
		err = h.processCheckoutSessionComplete(ctx, stripeEvent)
	case "invoice.paid":
		fallthrough
	case "invoice.payment_failed":
		err = h.processInvoice(ctx, stripeEvent)
	default:
		h.logger.Warn("unknown stripe webhook event", zap.String("type", stripeEvent.Type))
		return nil
	}
	if err != nil {
		h.logger.Error("while handling stripe event", zap.Error(err))
		return err
	}

	return nil
}

// errUnrecognizedMode indicates that a checkout event received was not
// produced in a checkout mode that this code is prepared to process.
var errUnrecognizedMode = errors.New("unrecognized checkout mode")

// processCheckoutSessionComplete handles a stripe "checkout.session.completed"
// event. CAUTION: If an error is returned, the event will not be acknowledged
// and will be reprocessed at a later time.
func (h Handler) processCheckoutSessionComplete(ctx context.Context, event stripe.Event) error {
	var checkout stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &checkout); err != nil {
		return fmt.Errorf("unmarshal checkout; error: %w", err)
	}

	var err error
	switch checkout.Mode {
	case stripe.CheckoutSessionModePayment:
		err = h.processPaymentCheckoutSessionComplete(ctx, event.ID, checkout)
	case stripe.CheckoutSessionModeSubscription:
		err = h.processSubscriptionCheckoutSessionComplete(ctx, event.ID, checkout)
	default:
		return fmt.Errorf("while processing checkout event: %w (%s)", errUnrecognizedMode, checkout.Mode)
	}

	return err
}

func (h Handler) processPaymentCheckoutSessionComplete(
	ctx context.Context,
	eventID string,
	checkout stripe.CheckoutSession,
) error {
	var errstr string
	switch {
	case checkout.ClientReferenceID == "":
		errstr = "checkout ClientReferenceID empty"
	case checkout.ID == "":
		errstr = "checkout ID empty"
	case checkout.Customer == nil:
		errstr = "checkout Customer nil"
	case checkout.Customer.ID == "":
		errstr = "checkout Customer ID empty"
	case checkout.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid:
		errstr = "checkout payment status is not \"paid\""
	case len(checkout.LineItems.Data) != 1:
		errstr = "checkout not for a single item"
	}
	if errstr != "" {
		// Log the error as a warning and return nil so that Stripe event is
		// discarded and not processed again.
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

	// Check to see if Stripe event ID exists in DB. If it does, this indicates
	// that event has already been processed.
	_, err = h.store.FirstVipByStripeEventID(ctx, eventID)
	if err == nil {
		// Checkout has already been processed, return early.
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrNotFound) {
		return fmt.Errorf("while finding checkout by stripe event ID: %w", err)
	}

	price := checkout.LineItems.Data[0].Price.ID

	// NOTE: It is possible that two processes simultaneously execute the below
	// CreateVip method. In the event this occurs, one will result an
	// error as two unique indexes will be violated: IdxVipsStripeEventID,
	// IdxCustomersSteamID, and IdxCustomersStripeCustomerID. If this is a
	// concern, consider introduce a transaction to ensure only one process may
	// check for and create a VIP.
	if err := h.store.CreateVip(
		ctx,
		&model.Vip{
			StripeCheckoutID: checkout.ID,
			StripeEventID:    eventID,
			ServerID:         stagedCheckout.ServerID,
			ExpiresAt:        model.ComputeVipExpiration(istripe.Price(price)),
		},
		&model.Customer{
			UserID:           stagedCheckout.UserID,
			StripeCustomerID: checkout.Customer.ID,
			SteamID:          stagedCheckout.SteamID,
		},
	); err != nil {
		return fmt.Errorf("while creating checkout: %w", err)
	}

	return h.invoicePaid(ctx, stagedCheckout.ServerID, stagedCheckout.SteamID)
}

func (h Handler) processSubscriptionCheckoutSessionComplete(
	ctx context.Context,
	eventID string,
	checkout stripe.CheckoutSession,
) error {
	var errstr string
	switch {
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
	case len(checkout.LineItems.Data) != 1:
		errstr = "checkout not for a single item"
	}
	if errstr != "" {
		// Log the error as a warning and return nil so that Stripe event is
		// discarded and not processed again.
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

	_, err = h.store.FirstVipByStripeEventID(ctx, eventID)
	if err == nil {
		// Checkout has already been processed, return early.
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrNotFound) {
		return fmt.Errorf("while finding vip by stripe event ID: %w", err)
	}

	price := checkout.LineItems.Data[0].Price.ID

	// NOTE: It is possible that two processes simultaneously execute the below
	// CreateSubscription method. In the event this occurs, one will result an
	// error as two unique indexes will be violated:
	// IdxSubscriptionsStripeSubscriptionID and IdxVipsStripeEventID. If
	// this is a concern, consider introduce a transaction to ensure only
	// one process may check for and create a subscription.
	if err := h.store.CreateVipSubscription(
		ctx,
		&model.Vip{
			StripeCheckoutID: checkout.ID,
			StripeEventID:    eventID,
			ServerID:         stagedCheckout.ServerID,
			ExpiresAt:        model.ComputeVipExpiration(istripe.Price(price)),
		},
		&model.Subscription{
			StripeSubscriptionID: checkout.Subscription.ID,
		},
		&model.Customer{
			UserID:           stagedCheckout.UserID,
			StripeCustomerID: checkout.Customer.ID,
			SteamID:          stagedCheckout.SteamID,
		},
	); err != nil {
		return fmt.Errorf("while creating vip subscription: %w", err)
	}

	return nil
}

func (h Handler) processInvoice(ctx context.Context, event stripe.Event) error {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	var errstr string
	switch {
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

	_, err := h.store.FirstInvoiceByStripeEventID(ctx, event.ID)
	if err == nil {
		// Invoice has already been processed, return early.
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrNotFound) {
		return fmt.Errorf("store.FindByStripeEventID: %w", err)
	}

	vip, err := h.store.AddInvoiceToVipSubscription(
		ctx,
		invoice.Subscription.ID,
		&model.Invoice{
			Status:        model.InvoiceStatus(string(invoice.Status)),
			StripeEventID: event.ID,
		},
	)
	if err != nil {
		return fmt.Errorf("while updating vip subscription invoices: %w", err)
	}

	// If invoice is anything other than "paid" return and do not publish an
	// invoice paid event.
	if invoice.Status != stripe.InvoiceStatusPaid {
		return nil
	}

	return h.invoicePaid(ctx, vip.Server.ID, vip.Customer.SteamID)
}

func (h Handler) invoicePaid(
	ctx context.Context,
	serverID uuid.UUID,
	steamID string,
) error {
	paid := event.NewInvoicePaidEvent(serverID, steamID)

	b, err := json.Marshal(&paid)
	if err != nil {
		return fmt.Errorf("while marshalling invoice paid event: %w", err)
	}

	if err := h.stream.Write(ctx, b); err != nil {
		return fmt.Errorf("while publishing invoice paid event: %w", err)
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
