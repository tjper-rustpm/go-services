// Package event provides types relevant to signal service changes outward
// to event consumers.
package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	case InvoicePaid:
		event = &InvoicePaidEvent{}
	case InvoicePaymentFailure:
		event = &InvoicePaymentFailureEvent{}
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
	InvoicePaid           Kind = "invoice_paid"
	InvoicePaymentFailure Kind = "invoice_payment_failure"
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

type InvoicePaidEvent struct {
	Event
	SubscriptionID uuid.UUID
	ServerID       uuid.UUID
	SteamID        string
}

func NewInvoicePaidEvent(
	subscriptionID uuid.UUID,
	serverID uuid.UUID,
	steamID string,
) InvoicePaidEvent {
	return InvoicePaidEvent{
		Event:          New(InvoicePaid),
		SubscriptionID: subscriptionID,
		ServerID:       serverID,
		SteamID:        steamID,
	}
}

type InvoicePaymentFailureEvent struct {
	Event
	SubscriptionID uuid.UUID
}

func NewInvoicePaymentFailure(subscriptionID uuid.UUID) InvoicePaymentFailureEvent {
	return InvoicePaymentFailureEvent{
		Event:          New(InvoicePaymentFailure),
		SubscriptionID: subscriptionID,
	}
}
