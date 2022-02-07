package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Subscription struct {
	model.Model
	ServerID uuid.UUID
	UserID   uuid.UUID

	StripeCheckoutID     string
	StripeCustomerID     string
	StripeSubscriptionID string `gorm:"uniqueIndex"`

	Event    Event     `gorm:"polymorphic:Owner;"`
	Invoices []Invoice `gorm:"foreignKey:StripeSubscriptionID;references:StripeSubscriptionID"`
}

type Invoice struct {
	model.Model

	StripeSubscriptionID string

	Event  Event `gorm:"polymorphic:Owner;"`
	Status InvoiceStatus
}

type InvoiceStatus string

const (
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusPaymentFailed InvoiceStatus = "payment_failed"
)

type Event struct {
	model.Model

	StripeEventID string `gorm:"uniqueIndex"`

	OwnerID   uuid.UUID
	OwnerType string
}

func (e Event) Exists(ctx context.Context, db *gorm.DB) (bool, error) {
	res := db.
		WithContext(ctx).
		Where("stripe_event_id = ?", e.StripeEventID).
		First(&e)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return false, fmt.Errorf("event exists; error: %w", res.Error)
	}
	return true, nil
}
