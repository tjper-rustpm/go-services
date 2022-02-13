// Package event provides types relevant to signal service changes outward
// to event consumers.
package event

import (
	"time"

	"github.com/google/uuid"
)

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
