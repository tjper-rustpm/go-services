package model

import (
	"context"
	"errors"
	"fmt"
	"time"

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

// Status retrieves the status of the subscription.
func (sub Subscription) Status() InvoiceStatus {
	if len(sub.Invoices) == 0 {
		return InvoiceStatusUnknown
	}

	latest := sub.Invoices[0]
	for _, invoice := range sub.Invoices {
		if invoice.CreatedAt.After(latest.CreatedAt) {
			latest = invoice
		}
	}

	duration := time.Hour * 24 * 30 // 30 days
	if latest.CreatedAt.Before(time.Now().Add(-duration)) {
		return InvoiceStatusInactive
	}

	return latest.Status
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
		return fmt.Errorf("while Find: %w", err)
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
		return fmt.Errorf("while First: %w", err)
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
	InvoiceStatusUnknown       InvoiceStatus = "unknown"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusPaymentFailed InvoiceStatus = "payment_failed"
	InvoiceStatusInactive      InvoiceStatus = "inactive"
)

type Server struct {
	ID                  uuid.UUID      `json:"id"`
	ActiveSubscriptions uint16         `gorm:"->" json:"activeSubscriptions"`
	SubscriptionLimit   uint16         `json:"subscriptionLimit"`
	Subscriptions       []Subscription `json:"-"`

	model.At
}

// Create creates the Server in the specified db. If the Server already exists,
// internal/gorm.ErrAlreadyExists is returned.
func (s *Server) Create(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.First(&Server{}, s.ID).Error
		if err == nil {
			return igorm.ErrAlreadyExists
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
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

type Servers []Server

// FindActiveSubscriptions retrieves each Servers subscription status from the
// specified db.
func (s *Servers) FindActiveSubscriptions(ctx context.Context, db *gorm.DB) error {
	sql := `
SELECT servers.id,
       COUNT(subscription_status.id) AS active_subscriptions,
       servers.subscription_limit,
       servers.updated_at,
       servers.created_at,
       servers.deleted_at
FROM payments.servers 
LEFT JOIN (
  SELECT subscriptions.id,
         subscriptions.server_id,
         invoices.status
  FROM payments.subscriptions
  JOIN payments.invoices
    ON invoices.subscription_id = subscriptions.id
  JOIN (
    SELECT sub_invoices.subscription_id,
           MAX(sub_invoices.created_at) as created_at
    FROM payments.invoices AS sub_invoices
    GROUP BY sub_invoices.subscription_id
  ) AS most_recent_invoices
    ON most_recent_invoices.subscription_id = invoices.subscription_id
       AND most_recent_invoices.created_at = invoices.created_at
  WHERE invoices.status = 'paid'
        AND invoices.created_at > now() - interval '31 days'
) AS subscription_status
  ON subscription_status.server_id = servers.id
GROUP BY servers.id
`

	if err := db.Raw(sql).Scan(s).Error; err != nil {
		return fmt.Errorf("while Scan: %w", err)
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

// IsSubscribedToServer checks if the customer has an active subscription to
// the specified server. The return values, are true - nil if a subscription
// exists, and false - nil if a subscription does not exist. Any error
// encountered is returned as the second return value.
func (c *Customer) IsSubscribedToServer(ctx context.Context, db *gorm.DB, serverID uuid.UUID) (bool, error) {
	sql := `
SELECT 1
FROM payments.subscriptions
WHERE subscriptions.customer_id = ?
      AND subscriptions.server_id = ?
      AND EXISTS (
        SELECT 1
        FROM payments.invoices
        JOIN (
          SELECT sub_invoices.subscription_id,
                 MAX(sub_invoices.created_at) as created_at
          FROM payments.invoices AS sub_invoices
          GROUP BY sub_invoices.subscription_id
        ) AS most_recent_invoices
          ON most_recent_invoices.subscription_id = invoices.subscription_id
             AND most_recent_invoices.created_at = invoices.created_at
        WHERE invoices.status = 'paid'
              AND invoices.created_at > now() - interval '31 days'
      )
`

	var exists bool
	if err := db.Raw(sql, c.UserID, serverID).Scan(&exists).Error; err != nil {
		return false, fmt.Errorf("model Customer.IsSubscribedToServer Scan: %w", err)
	}
	return exists, nil
}
