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

// Parse accepts a slice of bytes (b) and decodes these bytes into the
// appropriate event type.
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
	case VipRefresh:
		event = &VipRefreshEvent{}
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
	VipRefresh    Kind = "vip_refresh"
)

// New creates a new Event instance.
func New(kind Kind) Event {
	return Event{
		ID:        uuid.New(),
		Kind:      kind,
		CreatedAt: time.Now(),
	}
}

// Event is a generic Rustpm system event.
type Event struct {
	ID        uuid.UUID
	Kind      Kind
	CreatedAt time.Time
}

// StripeWebhookEvent is fired when a Stripe webhook event is available to
// processed.
type StripeWebhookEvent struct {
	Event
	StripeEvent stripe.Event
}

// NewStripeWebhookEvent creates a new StripeWebhookEvent instance.
func NewStripeWebhookEvent(stripeEvent stripe.Event) StripeWebhookEvent {
	return StripeWebhookEvent{
		Event:       New(StripeWebhook),
		StripeEvent: stripeEvent,
	}
}

// VipRefreshEvent is fired when a VIP within the Rustpm system has had its
// expiration refreshed.
type VipRefreshEvent struct {
	Event
	ServerID  uuid.UUID
	SteamID   string
	ExpiresAt time.Time
}

// NewVipRefreshEvent creates a new VipRefreshEvent instance.
func NewVipRefreshEvent(
	serverID uuid.UUID,
	steamID string,
	expiresAt time.Time,
) VipRefreshEvent {
	return VipRefreshEvent{
		Event:     New(VipRefresh),
		ServerID:  serverID,
		SteamID:   steamID,
		ExpiresAt: expiresAt,
	}
}
