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

	ServerID uuid.UUID
	Server   Server

	CustomerID uuid.UUID
	Customer   Customer `gorm:"foreignKey:UserID;references:CustomerID"`

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
// Server, this creation fails.
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

		sub.ServerID = serverID
		sub.CustomerID = customer.UserID

		if err := tx.Create(sub).Error; err != nil {
			return fmt.Errorf("create subscription; error: %w", err)
		}

		return nil
	})
}

// FirstByStripeEventID retrieves the Subscription entity based on the
// populated StripeEventID. If no Subscription is found,
// internal/gorm.ErrNotFound is returned.
func (sub *Subscription) FirstByStripeEventID(ctx context.Context, db *gorm.DB) error {
	err := db.WithContext(ctx).Where("stripe_event_id = ?", sub.StripeEventID).First(sub).Error
	if err != nil {
		return fmt.Errorf("find subscription by event ID; error: %w", err)
	}

	return nil
}

// Subscription many Subscription entities.
type Subscriptions []Subscription

// FindByUserID retrieves Subscriptions that belong the specified userID.
func (subs *Subscriptions) FindByUserID(ctx context.Context, db *gorm.DB, userID uuid.UUID) error {
	err := db.
		WithContext(ctx).
		Preload("Customer").
		Preload("Server").
		Preload("Invoices").
		Where("customer_id = ?", userID).
		Find(subs).Error
	if err != nil {
		return fmt.Errorf("Find: %w", err)
	}
	return nil
}

// First fetches the Subscription entity. If it is not found,
// internal/gorm.ErrNotFound is returned.
func (sub *Subscription) First(ctx context.Context, db *gorm.DB) error {
	err := db.
		WithContext(ctx).
		Preload("Customer").
		Preload("Server").
		Preload("Invoices").
		First(sub, sub.ID).Error
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

// FirstByStripeEventID retrieves the Invoice entity based on the populated
// StripeEventID. If no Invoice is found, internal/gorm.ErrNotFound is
// returned.
func (i *Invoice) FirstByStripeEventID(ctx context.Context, db *gorm.DB) error {
	err := db.WithContext(ctx).Where("stripe_event_id = ?", i.StripeEventID).First(i).Error
	if err != nil {
		return fmt.Errorf("First: %w", err)
	}
	return nil
}

type InvoiceStatus string

const (
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusPaymentFailed InvoiceStatus = "payment_failed"
)

type Server struct {
	ID                uuid.UUID
	SubscriptionLimit uint8
	Subscriptions     []Subscription

	model.At
}

// Create creates the Server in the specified db. If the Server already exists,
// internal/gorm.ErrAlreadyExists is returned.
func (s *Server) Create(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Find(ctx, &Server{}, s.ID).Error
		if err == nil {
			return igorm.ErrAlreadyExists
		}
		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Create(s).Error; err != nil {
			return err
		}

		return nil
	})
}

// First fetches the Server entity. If it is not found,
// internal/gorm.ErrNotFound is returned.
func (s *Server) First(ctx context.Context, db *gorm.DB) error {
	err := db.WithContext(ctx).First(s, s.ID).Error
	if err != nil {
		return fmt.Errorf("First: %w", err)
	}
	return nil
}

type Customer struct {
	UserID           uuid.UUID
	StripeCustomerID string
	SteamID          string
	Subscriptions    []Subscription `gorm:"foreignKey:CustomerID;references:UserID"`

	model.At
}

// First fetches the Customer entity. If it is not found,
// internal/gorm.ErrNotFound is returned.
func (c *Customer) First(ctx context.Context, db *gorm.DB) error {
	err := db.WithContext(ctx).First(c, c.UserID).Error
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
