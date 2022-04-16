package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/payment/model"
	igorm "github.com/tjper/rustcron/internal/gorm"

	"gorm.io/gorm"
)

// NewStore creates a Store instance.
func NewStore(db *gorm.DB) *Store {
	return &Store{
		Store: *igorm.NewStore(db),
		db:    db,
	}
}

// Store provides an API for interacting with the payment datastore.
type Store struct {
	igorm.Store
	db *gorm.DB
}

// CreateSubscription creates a subscription entity and its dependencies. The
// passed *model.Subscription is updated with the result.
func (s Store) CreateSubscription(
	ctx context.Context,
	sub *model.Subscription,
	customer *model.Customer,
	serverID uuid.UUID,
) error {
	return sub.Create(ctx, s.db, customer, serverID)
}

// CreateInvoice creates an invoice entity and its dependencies. The passed
// *model.Invoice is updated with the result.
func (s Store) CreateInvoice(
	ctx context.Context,
	invoice *model.Invoice,
	stripeSubscriptionID string,
) error {
	return invoice.Create(ctx, s.db, stripeSubscriptionID)
}

// FirstByStripeEventID encompasses a type that is able to retrieve itself
// from *gorm.DB by its Stripe event ID.
type FirsterByStripeEventID interface {
	FirstByStripeEventID(context.Context, *gorm.DB) error
}

// FirstByStripeEventID wraps execution of entity.FindByStripeEventID.
func (s Store) FirstByStripeEventID(ctx context.Context, entity FirsterByStripeEventID) error {
	return entity.FirstByStripeEventID(ctx, s.db)
}
