package model

import (
	"context"
	"errors"
	"fmt"

	igorm "github.com/tjper/rustcron/internal/gorm"
	"github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Subscription struct {
	model.Model

	StripeCheckoutID     string
	StripeSubscriptionID string
	StripeEventID        string

	ServerSubscriptionLimitID uuid.UUID
	ServerSubscriptionLimit   ServerSubscriptionLimit

	CustomerID uuid.UUID
	Customer   Customer

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

// Create creates the Subscription entity and its dependencies. If the passed
// Customer does not exist, it is created. If the serverID is not related to a
// ServerSubscriptionLimit, this creation fails.
func (sub *Subscription) Create(
	ctx context.Context,
	db *gorm.DB,
	customer *Customer,
	serverID uuid.UUID,
) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := customer.CreateIfStripeCustomerIDUnknown(ctx, tx); err != nil {
			return err
		}

		sub.ServerSubscriptionLimitID = serverID
		sub.CustomerID = customer.UserID

		if err := tx.Create(sub).Error; err != nil {
			return fmt.Errorf("create subscription; error: %w", err)
		}

		return nil
	})
}

// FindByStripeEventID retrieves the Subscription entity based on the
// populated StripeEventID. If no Subscription is found,
// internal/gorm.ErrNotFound is returned.
func (sub *Subscription) FindByStripeEventID(ctx context.Context, db *gorm.DB) error {
	err := db.WithContext(ctx).Where("stripe_event_id = ?", sub.StripeEventID).First(sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return igorm.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("find subscription by event ID; error: %w", err)
	}

	return nil
}

// First fetches the Subscription entity. If it is not found,
// internal/gorm.ErrNotFound is returned.
func (sub *Subscription) First(ctx context.Context, db *gorm.DB) error {
	err := db.
		WithContext(ctx).
		Preload("Customer").
		Preload("ServerSubscriptionLimit").
		Preload("Invoices").
		First(sub, sub.ID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return igorm.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("First: %w", err)
	}
	return nil
}

// Invoice is the record of a payment transaction.
type Invoice struct {
	model.Model

	SubscriptionID uuid.UUID
	StripeEventID  string

	Status InvoiceStatus
}

// Create creates the Invoice entity and relates it to its subscription. If the
// passes stripeSubscriptionID has not been processed, this creation fails.
func (i *Invoice) Create(ctx context.Context, db *gorm.DB, stripeSubscriptionID string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		subscription := &Subscription{}
		if err := tx.
			Where("stripe_subscription_id = ?", stripeSubscriptionID).
			First(subscription).Error; err != nil {
			return fmt.Errorf("First: %w", err)
		}

		i.SubscriptionID = subscription.ID

		if err := tx.Create(i).Error; err != nil {
			return fmt.Errorf("Create: %w", err)
		}

		return nil
	})
}

// FindByStripeEventID retrieves the Invoice entity based on the populated
// StripeEventID. If no Invoice is found, internal/gorm.ErrNotFound is
// returned.
func (i *Invoice) FindByStripeEventID(ctx context.Context, db *gorm.DB) error {
	err := db.WithContext(ctx).Where("stripe_event_id = ?", i.StripeEventID).First(i).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return igorm.ErrNotFound
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("find invoice by event ID; error: %w", err)
	}
	return nil
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

// First fetches the ServerSubscriptionLimit entity. If it is not found,
// internal/gorm.ErrNotFound is returned.
func (l *ServerSubscriptionLimit) First(ctx context.Context, db *gorm.DB) error {
	err := db.WithContext(ctx).First(l, l.ServerID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return igorm.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("first server subscription limit; error: %w", err)
	}
	return nil
}

type Customer struct {
	UserID           uuid.UUID
	StripeCustomerID string
	SteamID          string
	Subscriptions    []Subscription `gorm:"foreignKey:CustomerID"`

	model.At
}

// First fetches the Customer entity. If it is not found,
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

// CreateIfStripeCustomerIDUnknown creates the Customer entity if the
// StripeCustomerID is not associated with a Customer. If the StripeCustomerID
// is in use, the Customer is populated with the related data.
func (c *Customer) CreateIfStripeCustomerIDUnknown(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("stripe_customer_id = ?", c.StripeCustomerID).First(c).Error
		if err == nil {
			return nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("first customer w/ stripe customer ID; error: %w", err)
		}

		if err := tx.Create(c).Error; err != nil {
			return fmt.Errorf("create customer; error: %w", err)
		}

		return nil
	})
}
