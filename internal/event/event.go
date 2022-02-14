// Package event provides types relevant to signal service changes outward
// to event consumers.
package event

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/internal/hash"
)

func ParseHash(h map[string]interface{}) (interface{}, error) {
	var event interface{}
	switch kind := h["Kind"]; kind {
	case SubscriptionCreated:
		event = SubscriptionCreatedEvent{}
	case SubscriptionDeleted:
		event = SubscriptionDeleteEvent{}
	default:
		return nil, fmt.Errorf("unexpected event; kind: %s", kind)
	}

	if err := hash.ToStruct(&event, h); err != nil {
		return nil, fmt.Errorf("hash to struct; error: %w", err)
	}

	return event, nil
}

type Kind string

const (
	SubscriptionCreated Kind = "subscription_created"
	SubscriptionDeleted Kind = "subscription_deleted"
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

func NewSubscriptionCreatedEvent(
	subscriptionID uuid.UUID,
	userID uuid.UUID,
	serverID uuid.UUID,
) SubscriptionCreatedEvent {
	return SubscriptionCreatedEvent{
		Event:          New(SubscriptionCreated),
		SubscriptionID: subscriptionID,
		UserID:         userID,
		ServerID:       serverID,
	}
}

type SubscriptionCreatedEvent struct {
	Event
	SubscriptionID uuid.UUID
	UserID         uuid.UUID
	ServerID       uuid.UUID
}

func NewSubscriptionDeleteEvent(subscriptionID uuid.UUID) SubscriptionDeleteEvent {
	return SubscriptionDeleteEvent{
		Event:          New(SubscriptionDeleted),
		SubscriptionID: subscriptionID,
	}
}

type SubscriptionDeleteEvent struct {
	Event
	SubscriptionID uuid.UUID
}
