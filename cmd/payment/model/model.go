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

	StripeCheckoutID     string
	StripeCustomerID     string
	StripeSubscriptionID string
	StripeEventID        string

	Invoices []Invoice
}

func (sub Subscription) IsActive() bool {
	if len(sub.Invoices) < 1 {
		return false
	}

	latest := sub.Invoices[0]
	for _, invoice := range sub.Invoices {
		if invoice.CreatedAt.After(latest.CreatedAt) {
			latest = invoice
		}
	}

	return latest.Status == InvoiceStatusPaid
}

func (sub Subscription) ExistsWithStripeEventID(ctx context.Context, db *gorm.DB) (bool, error) {
	res := db.
		WithContext(ctx).
		Where("stripe_event_id = ?", sub.StripeEventID).
		First(&sub)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return false, fmt.Errorf("subscription exists with stripe event id; error: %w", res.Error)
	}
	return true, nil
}

type Invoice struct {
	model.Model

	SubscriptionID uuid.UUID
	StripeEventID  string

	Status InvoiceStatus
}

func (i Invoice) ExistsWithStripeEventID(ctx context.Context, db *gorm.DB) (bool, error) {
	res := db.
		WithContext(ctx).
		Where("stripe_event_id = ?", i.StripeEventID).
		First(&i)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return false, fmt.Errorf("invoice exists with stripe event id; error: %w", res.Error)
	}
	return true, nil
}

type InvoiceStatus string

const (
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusPaymentFailed InvoiceStatus = "payment_failed"
)
