package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/payment/model"
)

// NewStoreMock creates a new StoreMock instance.
func NewStoreMock(options ...StoreMockOption) *StoreMock {
	mock := &StoreMock{}

	for _, option := range options {
		option(mock)
	}

	return mock
}

// StoreMockOption is a function type that may configure a StoreMock instance.
type StoreMockOption func(*StoreMock)

// WithFirstServerByID configures a StoreMock instance to execute the passed
// function when FirstServerByID is called.
func WithFirstServerByID(fn firstServerByIDFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.firstServerByID = fn }
}

// WithFirstCustomerByUserID configures a StoreMock instance to execute the passed
// function when FirstCustomerByUserID is called.
func WithFirstCustomerByUserID(fn firstCustomerByUserIDFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.firstCustomerByUserID = fn }
}

// WithFirstSubscriptionByStripeEventID configures a StoreMock instance to execute the passed
// function when FirstSubscriptionByStripeEventID is called.
func WithFirstSubscriptionByStripeEventID(fn firstSubscriptionByStripeEventIDFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.firstSubscriptionByStripeEventID = fn }
}

// WithFirstInvoiceByStripeEventID configures a StoreMock instance to execute the passed
// function when FirstInvoiceByStripeEventID is called.
func WithFirstInvoiceByStripeEventID(fn firstInvoiceByStripeEventIDFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.firstInvoiceByStripeEventID = fn }
}

// WithFindSubscriptionsByUserID configures a StoreMock instance to execute the passed
// function when FindSubscriptionsByUserID is called.
func WithFindSubscriptionsByUserID(fn findSubscriptionsByUserIDFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.findSubscriptionsByUserID = fn }
}

// WithFindServers configures a StoreMock instance to execute the passed
// function when FindServers is called.
func WithFindServers(fn findServersFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.findServers = fn }
}

// WithCreateServer configures a StoreMock instance to execute the passed
// function when CreateServer is called.
func WithCreateServer(fn createServerFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.createServer = fn }
}

// WithCreateSubscription configures a StoreMock instance to execute the passed
// function when CreateSubscription is called.
func WithCreateSubscription(fn createSubscriptionFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.createSubscription = fn }
}

// WithCreateInvoice configures a StoreMock instance to execute the passed
// function when CreateInvoice is called.
func WithCreateInvoice(fn createInvoiceFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.createInvoice = fn }
}

// WithUpdateServer configures a StoreMock instance to execute the passed
// function when UpdateServer is called.
func WithUpdateServer(fn updateServerFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.updateServer = fn }
}

// WithIsCustomerSubscribed configures a StoreMock instance to execute the passed
// function when IsCustomerSubscribed is called.
func WithIsCustomerSubscribed(fn isCustomerSubscribedFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.isCustomerSubscribed = fn }
}

type (
	firstServerByIDFunc                  func(context.Context, uuid.UUID) (*model.Server, error)
	firstCustomerByUserIDFunc            func(context.Context, uuid.UUID) (*model.Customer, error)
	firstSubscriptionByStripeEventIDFunc func(context.Context, string) (*model.Subscription, error)
	firstInvoiceByStripeEventIDFunc      func(context.Context, string) (*model.Invoice, error)
	firstSubscriptionByIDFunc            func(context.Context, uuid.UUID) (*model.Subscription, error)
	findSubscriptionsByUserIDFunc        func(context.Context, uuid.UUID) (model.Subscriptions, error)
	findServersFunc                      func(context.Context) (model.Servers, error)
	createServerFunc                     func(context.Context, *model.Server) error
	createSubscriptionFunc               func(context.Context, *model.Subscription, *model.Customer) error
	createInvoiceFunc                    func(context.Context, *model.Invoice, string) error
	updateServerFunc                     func(context.Context, uuid.UUID, map[string]interface{}) (*model.Server, error)
	isCustomerSubscribedFunc             func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
)

// StoreMock is responsible for mocking payment store interactions. This type
// is typically used during unit-testing.
type StoreMock struct {
	firstServerByID                  firstServerByIDFunc
	firstCustomerByUserID            firstCustomerByUserIDFunc
	firstSubscriptionByStripeEventID firstSubscriptionByStripeEventIDFunc
	firstInvoiceByStripeEventID      firstInvoiceByStripeEventIDFunc
	firstSubscriptionByID            firstSubscriptionByIDFunc
	findSubscriptionsByUserID        findSubscriptionsByUserIDFunc
	findServers                      findServersFunc
	createServer                     createServerFunc
	createSubscription               createSubscriptionFunc
	createInvoice                    createInvoiceFunc
	updateServer                     updateServerFunc
	isCustomerSubscribed             isCustomerSubscribedFunc
}

// FirstServerByID calls the function configured via WithFirstServerID.
func (s StoreMock) FirstServerByID(ctx context.Context, id uuid.UUID) (*model.Server, error) {
	if s.firstServerByID == nil {
		return nil, errUnconfigured
	}
	return s.firstServerByID(ctx, id)
}

// FirstCustomerByUserID calls the function configured via
// WithFirstCustomerByUserID.
func (s StoreMock) FirstCustomerByUserID(ctx context.Context, userID uuid.UUID) (*model.Customer, error) {
	if s.firstCustomerByUserID == nil {
		return nil, errUnconfigured
	}
	return s.firstCustomerByUserID(ctx, userID)
}

// FirstSubscriptionByStripeEventID calls the function configured via
// WithFirstSubscriptionByStripeEventID.
func (s StoreMock) FirstSubscriptionByStripeEventID(ctx context.Context, stripeEventID string) (*model.Subscription, error) {
	if s.firstSubscriptionByStripeEventID == nil {
		return nil, errUnconfigured
	}
	return s.firstSubscriptionByStripeEventID(ctx, stripeEventID)
}

// FirstInvoiceByStripeEventID calls the function configured via
// WithFirstInvoiceByStripeEventID.
func (s StoreMock) FirstInvoiceByStripeEventID(ctx context.Context, stripeEventID string) (*model.Invoice, error) {
	if s.firstInvoiceByStripeEventID == nil {
		return nil, errUnconfigured
	}
	return s.firstInvoiceByStripeEventID(ctx, stripeEventID)
}

// FirstSubscriptionByID calls the function configured via
// WithFirstSubscriptionByID.
func (s StoreMock) FirstSubscriptionByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	if s.firstSubscriptionByID == nil {
		return nil, errUnconfigured
	}
	return s.firstSubscriptionByID(ctx, id)
}

// FindSubscriptionsByUserID calls the function configured via
// WithFindSubscriptionsByUserID.
func (s StoreMock) FindSubscriptionsByUserID(ctx context.Context, userID uuid.UUID) (model.Subscriptions, error) {
	if s.findSubscriptionsByUserID == nil {
		return nil, errUnconfigured
	}
	return s.findSubscriptionsByUserID(ctx, userID)
}

// FindServers calls the function configured via WithFindServers.
func (s StoreMock) FindServers(ctx context.Context) (model.Servers, error) {
	if s.findServers == nil {
		return nil, errUnconfigured
	}
	return s.findServers(ctx)
}

// CreateServer calls the function configured via WithCreateServer.
func (s StoreMock) CreateServer(ctx context.Context, server *model.Server) error {
	if s.createServer == nil {
		return errUnconfigured
	}
	return s.createServer(ctx, server)
}

// CreateSubscription calls the function configured via WithCreateSubscription.
func (s StoreMock) CreateSubscription(ctx context.Context, subscription *model.Subscription, customer *model.Customer) error {
	if s.createSubscription == nil {
		return errUnconfigured
	}
	return s.createSubscription(ctx, subscription, customer)
}

// CreateInvoice calls the function configured via WithCreateInvoice.
func (s StoreMock) CreateInvoice(ctx context.Context, invoice *model.Invoice, stripeSubscriptionID string) error {
	if s.createInvoice == nil {
		return errUnconfigured
	}
	return s.createInvoice(ctx, invoice, stripeSubscriptionID)
}

// UpdateServer calls the function configured via WithUpdateServer.
func (s StoreMock) UpdateServer(ctx context.Context, serverID uuid.UUID, changes map[string]interface{}) (*model.Server, error) {
	if s.updateServer == nil {
		return nil, errUnconfigured
	}
	return s.updateServer(ctx, serverID, changes)
}

// IsCustomerSubscribed calls the function configured via
// WithIsCustomerSubscribed.
func (s StoreMock) IsCustomerSubscribed(ctx context.Context, serverID, customerID uuid.UUID) (bool, error) {
	if s.isCustomerSubscribed == nil {
		return false, errUnconfigured
	}
	return s.isCustomerSubscribed(ctx, serverID, customerID)
}

// errUnconfigured indicates a call was made that the mock was not correctly
// configured for.
var errUnconfigured = errors.New("unconfigured mock call")
