package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/iancoleman/strcase"
	"github.com/tjper/rustcron/cmd/payment/model"
	igorm "github.com/tjper/rustcron/internal/gorm"
	"github.com/tjper/rustcron/internal/stripe"

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

// FirstCustomerBySteamID retrieves the customer with the specified steam ID.
// If the customer is not found, gorm.ErrNotFound is returned.
func (s Store) FirstCustomerBySteamID(ctx context.Context, steamID string) (*model.Customer, error) {
	var customer model.Customer
	err := s.db.WithContext(ctx).Where("steam_id = ?", steamID).First(&customer).Error
	if err != nil {
		return nil, fmt.Errorf("while retrieving first customer by steam id: %w", err)
	}
	return &customer, nil
}

// FirstVipByStripeEventID retrieves the vip with the passed Stripe event ID.
// If no vip is found, gorm.ErrNotFound is returned.
func (s Store) FirstVipByStripeEventID(ctx context.Context, stripeEventID string) (*model.Vip, error) {
	var vip model.Vip
	err := s.db.
		WithContext(ctx).
		Where("stripe_event_id = ?", stripeEventID).
		First(&vip).Error
	if err != nil {
		return nil, fmt.Errorf("while retrieving vip by stripe event ID: %w", err)
	}

	return &vip, nil
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

	// NOTE: Use make to initialize array so that even if SELECT returns zero rows,
	// this function returns an empty slice. Using var servers model.Servers below
	// declare the servers variable would result in null be returned in the event
	// SELECT returned zero rows.
	servers := make(model.Servers, 0)
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

// CreateVip creates the passed vip and the necessary dependencies.
func (s Store) CreateVip(
	ctx context.Context,
	vip *model.Vip,
	customer *model.Customer,
) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("stripe_customer_id = ?", customer.StripeCustomerID).First(customer).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("while retrieving customer by stripe customer ID: %w", err)
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(customer).Error; err != nil {
				return fmt.Errorf("while creating vip customer: %w", err)
			}
		}

		vip.CustomerID = customer.UserID
		if err := tx.Create(vip).Error; err != nil {
			return fmt.Errorf("while creating subscription vip: %w", err)
		}

		return nil
	})
}

// CreateVipSubscription creates the a VIP subscription and the necessary
// dependencies.
func (s Store) CreateVipSubscription(
	ctx context.Context,
	vip *model.Vip,
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
				return fmt.Errorf("while creating vip subscription customer: %w", err)
			}
		}

		vip.CustomerID = customer.UserID
		if err := tx.Create(vip).Error; err != nil {
			return fmt.Errorf("while creating subscription vip: %w", err)
		}

		subscription.VipID = vip.ID
		if err := tx.Create(subscription).Error; err != nil {
			return fmt.Errorf("while creating vip subscription: %w", err)
		}

		return nil
	})
}

// AddInvoiceToVipSubscription adds the passed invoice to the vip subscription
// associated with the passed Stripe subscription ID. The vip expiration that
// is updated and returned to the caller.
func (s Store) AddInvoiceToVipSubscription(
	ctx context.Context,
	stripeSubscriptionID string,
	invoice *model.Invoice,
) (*model.Vip, error) {
	var vip model.Vip
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var subscription model.Subscription
		if err := tx.
			Where("stripe_subscription_id = ?", stripeSubscriptionID).
			First(&subscription).Error; err != nil {
			return fmt.Errorf("while retrieving invoice subscription: %w", err)
		}

		invoice.SubscriptionID = subscription.ID
		if err := tx.Create(invoice).Error; err != nil {
			return fmt.Errorf("while creating vip subscription invoice: %w", err)
		}

		if err := tx.First(&vip, subscription.VipID); err != nil {
			return fmt.Errorf("while retrieving subscription vip: %w", err)
		}

		expiresAt := model.ComputeVipExpiration(stripe.MonthlyVipSubscription)
		vip.ExpiresAt = expiresAt

		if err := tx.Model(&vip).Update("expires_at", expiresAt).Error; err != nil {
			return fmt.Errorf("while updating subscription vip: %w", err)
		}
		return nil
	})

	return &vip, err
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

// IsServerVipBySteamID checks if the steam ID is an active vip on the the
// specified server. The return values, are true - nil if a subscription
// exists, and false - nil if a subscription does not exist. Any error
// encountered is returned as the second return value.
func (s Store) IsServerVipBySteamID(
	ctx context.Context,
	serverID uuid.UUID,
	steamID string,
) (bool, error) {
	sql := `
SELECT 1
FROM payments.vips
WHERE vips.server_id = ?
      AND vips.expires_at >= now()
      AND EXISTS (
        SELECT 1
        FROM payments.customers
        WHERE customers.user_id = vips.customer_id
              AND customers.steam_id = ?
      )
`

	var isVip bool
	if err := s.db.WithContext(ctx).Raw(sql, serverID, steamID).Scan(&isVip).Error; err != nil {
		return false, fmt.Errorf("while checking if steam ID has active subscription: %w", err)
	}

	return isVip, nil
}
