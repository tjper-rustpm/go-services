package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/tjper/rustcron/internal/model"
	igorm "github.com/tjper/rustcron/internal/gorm"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Subscription struct {
	model.Model

	StripeCheckoutID     string
	StripeSubscriptionID string
	StripeEventID        string

	ServerSubscriptionLimitID uuid.UUID
	CustomerID                uuid.UUID

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

type ServerSubscriptionLimit struct {
	ServerID      uuid.UUID
	Maximum       uint8
	Subscriptions []Subscription `gorm:"foreignKey:ServerSubscriptionLimitID"`

	model.At
}

// Create creates the ServerSubscriptionLimit in the specified db. If the 
// ServerSubscriptionLimit already exists, the internal/gorm.ErrAlreadyExists 
// is returned.
func (l *ServerSubscriptionLimit) Create(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Find(ctx, &ServerSubscriptionLimit{}, l.ServerID).Error
		if err == nil {
			return igorm.ErrAlreadyExists
		}
		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Create(l).Error; err != nil {
			return err
		}

		return nil
	})
}

type Customer struct {
	UserID           uuid.UUID
	StripeCustomerID string
	Subscriptions    []Subscription `gorm:"foreignKey:CustomerID"`

	model.At
}

// First fetches the Customer entity. If it not found, 
// internal/gorm.ErrNotFound is returned.
func (c *Customer) First(ctx context.Context, db *gorm.DB) error {
  err := db.WithContext(ctx).First(c, c.UserID).Error
  if errors.Is(err, gorm.ErrRecordNotFound) {
    return igorm.ErrNotFound 
  }
  if err != nil {
    return fmt.Errorf("first customer; error: %w", err)
  }

  return nil
}
