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

// FindVipsByUserID retrieves the vips with specified user ID.
func (s Store) FindVipsByUserID(ctx context.Context, userID uuid.UUID) (model.Vips, error) {
	var vips model.Vips
	err := s.db.WithContext(ctx).Where("customer_id = ?", userID).Find(&vips).Error
	if err != nil {
		return nil, fmt.Errorf("while finding vips by customer ID: %w", err)
	}
	return vips, nil
}

// FindServers retrieves all servers.
func (s Store) FindServers(ctx context.Context) (model.Servers, error) {
	sql := `
SELECT servers.id,
       COUNT(vips.id) AS active_subscriptions,
       servers.subscription_limit,
       servers.updated_at,
       servers.created_at,
       servers.deleted_at
FROM payments.servers 
LEFT JOIN payments.vips
  ON vips.server_id = servers.id
WHERE vips.expires_at > now()
      AND servers.deleted_at IS NULL
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
		if err := createCustomer(ctx, tx, customer); err != nil {
			return err
		}

		vip.CustomerID = customer.ID
		if err := tx.Create(vip).Error; err != nil {
			return fmt.Errorf("while creating vip: %w", err)
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
	user *model.User,
) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := createCustomer(ctx, tx, customer); err != nil {
			return err
		}

		user.CustomerID = customer.ID
		if err := createUser(ctx, tx, user); err != nil {
			return err
		}

		vip.CustomerID = customer.ID
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

		if err := tx.First(&vip, subscription.VipID).Error; err != nil {
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

// IsServerVipBySteamID checks if the steam ID is an active vip on the
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
      AND vips.expires_at > now()
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

// createCustomer inserts the customer into the passed db. If the customer
// already exists the passed customer is updated with the contents of the db
// and a nil error is returned.
func createCustomer(ctx context.Context, db *gorm.DB, customer *model.Customer) error {
	err := db.
		WithContext(ctx).
		Where("stripe_customer_id = ?", customer.StripeCustomerID).
		First(customer).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("while retrieving customer by stripe customer ID: %w", err)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.WithContext(ctx).Create(customer).Error; err != nil {
			return fmt.Errorf("while creating customer: %w", err)
		}
	}

	return nil
}

// createUser interst the user into the passed db. If the user already exists
// the passed user instance is updated with the details of the found user and a
// nil error is returned.
func createUser(ctx context.Context, db *gorm.DB, user *model.User) error {
	err := db.WithContext(ctx).First(user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("while retrieving user by ID: %w", err)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.WithContext(ctx).Create(user).Error; err != nil {
			return fmt.Errorf("while creating user: %w", err)
		}
	}

	return nil
}
