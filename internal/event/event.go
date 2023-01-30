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

	str, ok := m["kind"].(string)
	if !ok {
		return nil, errKindInvalid
	}

	var event interface{}
	switch Kind(str) {
	case StripeWebhook:
		event = &StripeWebhookEvent{}
	case VipRefresh:
		event = &VipRefreshEvent{}
	case ServerStatusChange:
		event = &ServerStatusChangeEvent{}
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
	StripeWebhook      Kind = "stripe_webhook"
	VipRefresh         Kind = "vip_refresh"
	ServerStatusChange Kind = "server_status_change"
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
	ID        uuid.UUID `json:"id"`
	Kind      Kind      `json:"kind"`
	CreatedAt time.Time `json:"createdAt"`
}

// StripeWebhookEvent is fired when a Stripe webhook event is available to
// processed.
type StripeWebhookEvent struct {
	Event
	StripeEvent stripe.Event `json:"stripeEvent"`
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
	ServerID  uuid.UUID `json:"serverId"`
	SteamID   string    `json:"steamId"`
	ExpiresAt time.Time `json:"expiresAt"`
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

type ServerStatus string

const (
	Live    ServerStatus = "live"
	Offline ServerStatus = "offline"
)

type ServerDetails struct {
	Status        ServerStatus `json:"status,omitempty"`
	ActivePlayers int          `json:"activePlayers"`
	MaxPlayers    int          `json:"maxPlayers"`

	// Mask constains the fields that have new values within the ServerDetails
	// instance.
	Mask []string `json:"mask"`
}

// ServerStatusChangeEvent is fired when a Rustpm server's status changes.
type ServerStatusChangeEvent struct {
	Event
	ServerID uuid.UUID     `json:"serverId"`
	Details  ServerDetails `json:"details"`
}

// NewServerStatusChangeEvent creates a new ServerStatusEvent instance.
func NewServerStatusChangeEvent(serverID uuid.UUID, changes ...ServerChange) ServerStatusChangeEvent {
	event := ServerStatusChangeEvent{
		Event:    New(ServerStatusChange),
		ServerID: serverID,
		Details: ServerDetails{
			Mask: make([]string, 0),
		},
	}

	for _, change := range changes {
		change(&event)
	}

	return event
}

// ServerChange is a function type encompasses implementations that modify the
// ServerStatusEvent to include details of a server change.
type ServerChange func(*ServerStatusChangeEvent)

// WithStatusChange updates a ServerStatusEvent to have the status specified.
func WithStatusChange(status ServerStatus) ServerChange {
	return func(e *ServerStatusChangeEvent) {
		e.Details.Status = status
		e.Details.Mask = append(e.Details.Mask, "status")
	}
}

// WithActivePlayers updates a ServerStatusEvent to have the number of active
// players specified.
func WithActivePlayers(activePlayers int) ServerChange {
	return func(e *ServerStatusChangeEvent) {
		e.Details.ActivePlayers = activePlayers
		e.Details.Mask = append(e.Details.Mask, "activePlayers")
	}
}

// WithMaxPlayers updates a ServerStatusEvent to have the number of max players
// specified.
func WithMaxPlayers(maxPlayers int) ServerChange {
	return func(e *ServerStatusChangeEvent) {
		e.Details.MaxPlayers = maxPlayers
		e.Details.Mask = append(e.Details.Mask, "maxPlayers")
	}
}
