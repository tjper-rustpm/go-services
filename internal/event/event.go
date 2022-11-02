// Package event provides types relevant to signal service changes outward
// to event consumers.
package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v72"
)

var errKindInvalid = errors.New("kind is not string type")

func Parse(b []byte) (interface{}, error) {
	m := make(map[string]interface{})
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshal event; error: %w", err)
	}

	str, ok := m["Kind"].(string)
	if !ok {
		return nil, errKindInvalid
	}

	var event interface{}
	switch Kind(str) {
	case StripeWebhook:
		event = &StripeWebhookEvent{}
	case InvoicePaid:
		event = &InvoicePaidEvent{}
	default:
		return nil, fmt.Errorf("unexpected event; kind: %s, error: %w", str, errKindInvalid)
	}

	if err := json.Unmarshal(b, event); err != nil {
		return nil, fmt.Errorf("unmarshal event; type: %T, error: %w", event, err)
	}

	return event, nil
}

type Kind string

const (
	StripeWebhook Kind = "stripe_webhook"
	InvoicePaid   Kind = "invoice_paid"
)

func New(kind Kind) Event {
	return Event{
		ID:        uuid.New(),
		Kind:      kind,
		CreatedAt: time.Now(),
	}
}

type Event struct {
	ID        uuid.UUID
	Kind      Kind
	CreatedAt time.Time
}

type StripeWebhookEvent struct {
	Event
	StripeEvent stripe.Event
}

func NewStripeWebhookEvent(stripeEvent stripe.Event) StripeWebhookEvent {
	return StripeWebhookEvent{
		Event:       New(StripeWebhook),
		StripeEvent: stripeEvent,
	}
}

// TODO: Update InvoicePaidEvent to CheckoutCompleteEvent.
type InvoicePaidEvent struct {
	Event
	ServerID uuid.UUID
	SteamID  string
}

func NewInvoicePaidEvent(
	serverID uuid.UUID,
	steamID string,
) InvoicePaidEvent {
	return InvoicePaidEvent{
		Event:    New(InvoicePaid),
		ServerID: serverID,
		SteamID:  steamID,
	}
}
