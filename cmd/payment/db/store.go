package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/iancoleman/strcase"
	"github.com/tjper/rustcron/cmd/payment/model"
	igorm "github.com/tjper/rustcron/internal/gorm"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NewStore creates a new Store instance.
func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// Store is responsible for payment store interactions.
type Store struct {
	db *gorm.DB
}

// FirstServerByID retrieves the server with the specified id. If the server is
// not found, gorm.ErrNotFound is returned.
func (s Store) FirstServerByID(ctx context.Context, id uuid.UUID) (*model.Server, error) {
	var server model.Server
	err := s.db.WithContext(ctx).First(&server, id).Error
	if err != nil {
		return nil, fmt.Errorf("while retrieving first server by id: %w", err)
	}
	return &server, nil
}

// FirstCustomerByUserID retrieves the customer with the specified user id. If
// the customer is not found, gorm.ErrNotFound is returned.
func (s Store) FirstCustomerByUserID(ctx context.Context, userID uuid.UUID) (*model.Customer, error) {
	var customer model.Customer
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&customer).Error
	if err != nil {
		return nil, fmt.Errorf("while retrieving first customer by user id: %w", err)
	}
	return &customer, nil
}

// FirstSubscriptionByStripeEventID retrieves the subscription with the passed
// stripe event ID. If no subscription is found, gorm.ErrNotFound is returned.
func (s Store) FirstSubscriptionByStripeEventID(ctx context.Context, stripeEventID string) (*model.Subscription, error) {
	var subscription model.Subscription
	err := s.db.
		WithContext(ctx).
		Where("stripe_event_id = ?", stripeEventID).
		First(&subscription).Error
	if err != nil {
		return nil, fmt.Errorf("while retrieving subscription by stripe event ID: %w", err)
	}

	return &subscription, nil
}

// FirstInvoiceByStripeEventID retrieves the invoice with the passed stripe
// event ID. If no invoice is found, gorm.ErrNotFound is returned.
func (s Store) FirstInvoiceByStripeEventID(ctx context.Context, stripeEventID string) (*model.Invoice, error) {
	var invoice model.Invoice
	err := s.db.
		WithContext(ctx).
		Where("stripe_event_id = ?", stripeEventID).
		First(&invoice).Error
	if err != nil {
		return nil, fmt.Errorf("while retrieving invoice by stripe event ID: %w", err)
	}

	return &invoice, nil
}

// FirstSubscriptionByID retrieves the subscription with the specified id. If
// it is not found, gorm.ErrNotFound is returned.
func (s Store) FirstSubscriptionByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	var subscription model.Subscription
	err := s.db.
		WithContext(ctx).
		Preload("Customer").
		Preload("Server").
		Preload("Invoices").
		First(&subscription, id).Error
	if err != nil {
		return nil, fmt.Errorf("while retrieving subscription by id: %w", err)
	}
	return &subscription, nil
}

// FindSubscriptionsByUserID retrieves the subscriptions with the specified
// user ID.
func (s Store) FindSubscriptionsByUserID(ctx context.Context, userID uuid.UUID) (model.Subscriptions, error) {
	var subscriptions model.Subscriptions
	err := s.db.
		WithContext(ctx).
		Preload("Customer").
		Preload("Server").
		Preload("Invoices").
		Where("customer_id = ?", userID).
		Find(&subscriptions).Error
	if err != nil {
		return nil, fmt.Errorf("while finding subscriptions by user ID: %w", err)
	}
	return subscriptions, nil
}

// FindServers retrieves all servers.
func (s Store) FindServers(ctx context.Context) (model.Servers, error) {
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

	var servers model.Servers
	if err := s.db.WithContext(ctx).Raw(sql).Scan(&servers).Error; err != nil {
		return nil, fmt.Errorf("while finding servers: %w", err)
	}
	return servers, nil
}

// CreateServer creates the passed server in the store. If a server with the
// same identifier already exists, gorm.ErrAlreadyExists is returned.
func (s Store) CreateServer(ctx context.Context, server *model.Server) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.First(server, server.ID).Error
		if err == nil {
			return igorm.ErrAlreadyExists
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Create(server).Error; err != nil {
			return err
		}

		return nil
	})
}

// CreateSubscription creates the passed subscription and its dependencies if
// necessary. If the customer does not exist within the store, it will be
// created.
func (s Store) CreateSubscription(
	ctx context.Context,
	subscription *model.Subscription,
	customer *model.Customer,
) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("stripe_customer_id = ?", customer.StripeCustomerID).First(customer).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("while retrieving customer by stripe customer ID: %w", err)
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(customer).Error; err != nil {
				return fmt.Errorf("while create customer; error: %w", err)
			}
		}

		subscription.CustomerID = customer.UserID

		if err := tx.Create(subscription).Error; err != nil {
			return fmt.Errorf("create subscription; error: %w", err)
		}

		return nil
	})
}

// CreateInvoice creates the invoice if a subscription with the specified
// stripe subscription ID exists.
func (s Store) CreateInvoice(ctx context.Context, invoice *model.Invoice, stripeSubscriptionID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var subscription model.Subscription
		if err := tx.
			Where("stripe_subscription_id = ?", stripeSubscriptionID).
			First(&subscription).Error; err != nil {
			return fmt.Errorf("while retrieving invoice subscription: %w", err)
		}

		invoice.SubscriptionID = subscription.ID

		if err := tx.Create(invoice).Error; err != nil {
			return fmt.Errorf("while creating invoice: %w", err)
		}

		return nil
	})
}

// UpdateServer ensures a server with the specified serverID exists, and
// updates the server with passed changes.
func (s Store) UpdateServer(ctx context.Context, serverID uuid.UUID, changes map[string]interface{}) (*model.Server, error) {
	snakeCaseChanges := make(map[string]interface{})
	for field, value := range changes {
		snakeCaseChanges[strcase.ToSnake(field)] = value
	}

	var server model.Server
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&server, serverID).Error; err != nil {
			return fmt.Errorf("while retrieving server to update: %w", err)
		}

		if err := tx.Model(&server).Updates(snakeCaseChanges).Error; err != nil {
			return fmt.Errorf("while updating server: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &server, nil
}

// IsCustomerSubscribed checks if the customer has an active subscription to
// the specified server. The return values, are true - nil if a subscription
// exists, and false - nil if a subscription does not exist. Any error
// encountered is returned as the second return value.
func (s Store) IsCustomerSubscribed(ctx context.Context, serverID, customerID uuid.UUID) (bool, error) {
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
	if err := s.db.WithContext(ctx).Raw(sql, customerID, serverID).Scan(&exists).Error; err != nil {
		return false, fmt.Errorf("while checking customer is subscribed: %w", err)
	}
	return exists, nil
}
