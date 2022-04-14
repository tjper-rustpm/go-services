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
	case SubscriptionCreated:
		event = &SubscriptionCreatedEvent{}
	case SubscriptionDeleted:
		event = &SubscriptionDeleteEvent{}
	case CustomerCreated:
		event = &CustomerCreatedEvent{}
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
	SubscriptionCreated Kind = "subscription_created"
	SubscriptionDeleted Kind = "subscription_deleted"
	CustomerCreated     Kind = "customer_created"
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

func NewCustomerCreatedEvent(userID uuid.UUID, customerID string) CustomerCreatedEvent {
	return CustomerCreatedEvent{
		Event:      New(CustomerCreated),
		UserID:     userID,
		CustomerID: customerID,
	}
}

type CustomerCreatedEvent struct {
	Event
	UserID     uuid.UUID
	CustomerID string
}
