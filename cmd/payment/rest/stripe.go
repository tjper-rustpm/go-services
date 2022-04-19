package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/internal/event"
	"github.com/tjper/rustcron/internal/gorm"
	ihttp "github.com/tjper/rustcron/internal/http"
	imodel "github.com/tjper/rustcron/internal/model"

	"github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
)

type Stripe struct {
	API
}

func (ep Stripe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	event, err := ep.stripe.ConstructEvent(
		b,
		r.Header.Get("Stripe-Signature"),
	)
	if err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	var handler func(context.Context, stripe.Event) error
	switch event.Type {
	case "checkout.session.completed":
		handler = ep.processCheckoutSessionComplete
	case "invoice.paid":
		fallthrough
	case "invoice.payment_failed":
		handler = ep.processInvoice
	default:
		ep.logger.Error("unknown webhook event", zap.String("type", event.Type))
		return
	}

	w.WriteHeader(http.StatusOK)

	err = handler(r.Context(), event)
	if err != nil {
		ep.logger.Error("stripe event handling", zap.Error(err))
	}
}

func (ep Stripe) processCheckoutSessionComplete(ctx context.Context, event stripe.Event) error {
	var checkout stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &checkout); err != nil {
		return fmt.Errorf("unmarshal checkout; error: %w", err)
	}

	stagedCheckout, err := ep.staging.FetchCheckout(ctx, checkout.ClientReferenceID)
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

	err = ep.store.FirstByStripeEventID(ctx, subscription)
	if err == nil {
		// Subscription has already been processed, return early.
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrNotFound) {
		return fmt.Errorf("find subscription be event ID; error: %w", err)
	}

	if err := ep.store.CreateSubscription(
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

func (ep Stripe) processInvoice(ctx context.Context, event stripe.Event) error {
	var invoiceEvent stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoiceEvent); err != nil {
		return fmt.Errorf("Unmarshal: %w", err)
	}

	invoice := &model.Invoice{
		Status:        model.InvoiceStatus(string(invoiceEvent.Status)),
		StripeEventID: event.ID,
	}

	err := ep.store.FirstByStripeEventID(ctx, invoice)
	if err == nil {
		// Invoice has already been processed, return early.
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrNotFound) {
		return fmt.Errorf("FindByStripeEventID: %w", err)
	}

	if err := ep.store.CreateInvoice(ctx, invoice, invoiceEvent.Subscription.ID); err != nil {
		return fmt.Errorf("CreateInvoice: %w", err)
	}

	subscription := &model.Subscription{
		Model: imodel.Model{ID: invoice.SubscriptionID},
	}

	if invoiceEvent.Status == stripe.InvoiceStatusPaid {
		if err = ep.invoicePaid(
			ctx,
			subscription.ID,
			subscription.Server.ID,
			subscription.Customer.SteamID,
		); err != nil {
			return fmt.Errorf("invoicePaid: %w", err)
		}
		return nil
	}

	if err = ep.invoicePaymentFailure(ctx, subscription.ID); err != nil {
		return fmt.Errorf("invoicePaymentFailure: %w", err)
	}
	return nil
}

func (ep Stripe) invoicePaid(
	ctx context.Context,
	subsriptionID uuid.UUID,
	serverID uuid.UUID,
	steamID string,
) error {
	paid := event.NewInvoicePaidEvent(subsriptionID, serverID, steamID)

	b, err := json.Marshal(&paid)
	if err != nil {
		return fmt.Errorf("Marshal: %w", err)
	}

	if err := ep.stream.Write(ctx, b); err != nil {
		return fmt.Errorf("Write: %w", err)
	}
	return nil
}

func (ep Stripe) invoicePaymentFailure(
	ctx context.Context,
	subscriptionID uuid.UUID,
) error {
	paymentFailure := event.NewInvoicePaymentFailure(subscriptionID)

	b, err := json.Marshal(&paymentFailure)
	if err != nil {
		return fmt.Errorf("Marshal: %w", err)
	}

	if err := ep.stream.Write(ctx, b); err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	return nil
}
