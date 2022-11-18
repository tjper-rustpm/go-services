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

// WithFirstVipByStripeEventID configures a StoreMock instance to execute the
// passed function when FirstVipByStripeEventID is called.
func WithFirstVipByStripeEventID(fn firstVipByStripeEventIDFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.firstVipByStripeEventID = fn }
}

// WithFirstInvoiceByStripeEventID configures a StoreMock instance to execute the passed
// function when FirstInvoiceByStripeEventID is called.
func WithFirstInvoiceByStripeEventID(fn firstInvoiceByStripeEventIDFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.firstInvoiceByStripeEventID = fn }
}

// WithFindVipsByUserID configures a StoreMock instance to execute the passed
// function when FindVipsByUserID is called.
func WithFindVipsByUserID(fn findVipsByUserIDFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.findVipsByUserID = fn }
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

// WithCreateVip configures a StoreMock instance to execute the passed
// function when CreateVip is called.
func WithCreateVip(fn createVipFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.createVip = fn }
}

// WithCreateVipSubscription configures a StoreMock instance to execute the
// passed function when CreateVipSubscription is called.
func WithCreateVipSubscription(fn createVipSubscriptionFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.createVipSubscription = fn }
}

// WithUpdateServer configures a StoreMock instance to execute the passed
// function when UpdateServer is called.
func WithUpdateServer(fn updateServerFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.updateServer = fn }
}

// WithIsServerVipBySteamID configures a StoreMock instance to execute the
// passed function when IsServerVipBySteamID is called.
func WithIsServerVipBySteamID(fn isServerVipBySteamIDFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.isServerVipBySteamID = fn }
}

// WithAddInvoiceToVipSubscription configres a StoreMock instance to execute
// the passed function when AddInvoiceToVipSubscription is called.
func WithAddInvoiceToVipSubscription(fn addInvoiceToVipSubscriptionFunc) StoreMockOption {
	return func(mock *StoreMock) { mock.addInvoiceToVipSubscription = fn }
}

type (
	firstServerByIDFunc             func(context.Context, uuid.UUID) (*model.Server, error)
	firstCustomerByUserIDFunc       func(context.Context, uuid.UUID) (*model.Customer, error)
	firstVipByStripeEventIDFunc     func(context.Context, string) (*model.Vip, error)
	firstInvoiceByStripeEventIDFunc func(context.Context, string) (*model.Invoice, error)
	findVipsByUserIDFunc            func(context.Context, uuid.UUID) (model.Vips, error)
	findServersFunc                 func(context.Context) (model.Servers, error)
	createServerFunc                func(context.Context, *model.Server) error
	createVipFunc                   func(context.Context, *model.Vip, *model.Customer) error
	createVipSubscriptionFunc       func(context.Context, *model.Vip, *model.Subscription, *model.Customer, *model.User) error
	addInvoiceToVipSubscriptionFunc func(context.Context, string, *model.Invoice) (*model.Vip, error)
	updateServerFunc                func(context.Context, uuid.UUID, map[string]interface{}) (*model.Server, error)
	isServerVipBySteamIDFunc        func(context.Context, uuid.UUID, string) (bool, error)
)

// StoreMock is responsible for mocking payment store interactions. This type
// is typically used during unit-testing.
type StoreMock struct {
	firstServerByID             firstServerByIDFunc
	firstCustomerByUserID       firstCustomerByUserIDFunc
	firstVipByStripeEventID     firstVipByStripeEventIDFunc
	firstInvoiceByStripeEventID firstInvoiceByStripeEventIDFunc
	findVipsByUserID            findVipsByUserIDFunc
	findServers                 findServersFunc
	createServer                createServerFunc
	createVip                   createVipFunc
	createVipSubscription       createVipSubscriptionFunc
	addInvoiceToVipSubscription addInvoiceToVipSubscriptionFunc
	updateServer                updateServerFunc
	isServerVipBySteamID        isServerVipBySteamIDFunc
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

// FirstVipByStripeEventID calls the function configured via
// WithFirstVipByStripeEventID.
func (s StoreMock) FirstVipByStripeEventID(ctx context.Context, stripeEventID string) (*model.Vip, error) {
	if s.firstVipByStripeEventID == nil {
		return nil, errUnconfigured
	}
	return s.firstVipByStripeEventID(ctx, stripeEventID)
}

// FirstInvoiceByStripeEventID calls the function configured via
// WithFirstInvoiceByStripeEventID.
func (s StoreMock) FirstInvoiceByStripeEventID(ctx context.Context, stripeEventID string) (*model.Invoice, error) {
	if s.firstInvoiceByStripeEventID == nil {
		return nil, errUnconfigured
	}
	return s.firstInvoiceByStripeEventID(ctx, stripeEventID)
}

// FindVipsByUserID calls the function configured via WithFindVipsByUserID.
func (s StoreMock) FindVipsByUserID(ctx context.Context, userID uuid.UUID) (model.Vips, error) {
	if s.findVipsByUserID == nil {
		return nil, errUnconfigured
	}
	return s.findVipsByUserID(ctx, userID)
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

// CreateVip calls the function configured via WithCreateVip.
func (s StoreMock) CreateVip(ctx context.Context, vip *model.Vip, customer *model.Customer) error {
	if s.createVip == nil {
		return errUnconfigured
	}
	return s.createVip(ctx, vip, customer)
}

// CreateVipSubscription calls the function configured via WithCreateVipSubscription.
func (s StoreMock) CreateVipSubscription(ctx context.Context, vip *model.Vip, subscription *model.Subscription, customer *model.Customer, user *model.User) error {
	if s.createVipSubscription == nil {
		return errUnconfigured
	}
	return s.createVipSubscription(ctx, vip, subscription, customer, user)
}

// AddInvoiceToVipSubscription calls the function configured via
// WithAddInvoiceToVipSubscription.
func (s StoreMock) AddInvoiceToVipSubscription(ctx context.Context, stripeSubscriptionID string, invoice *model.Invoice) (*model.Vip, error) {
	if s.addInvoiceToVipSubscription == nil {
		return nil, errUnconfigured
	}
	return s.addInvoiceToVipSubscription(ctx, stripeSubscriptionID, invoice)
}

// UpdateServer calls the function configured via WithUpdateServer.
func (s StoreMock) UpdateServer(ctx context.Context, serverID uuid.UUID, changes map[string]interface{}) (*model.Server, error) {
	if s.updateServer == nil {
		return nil, errUnconfigured
	}
	return s.updateServer(ctx, serverID, changes)
}

// IsServerVipBySteamID calls the function configured via
// WithIsServerVipBySteamID.
func (s StoreMock) IsServerVipBySteamID(ctx context.Context, serverID uuid.UUID, steamID string) (bool, error) {
	if s.isServerVipBySteamID == nil {
		return false, errUnconfigured
	}
	return s.isServerVipBySteamID(ctx, serverID, steamID)
}

// errUnconfigured indicates a call was made that the mock was not correctly
// configured for.
var errUnconfigured = errors.New("unconfigured mock call")
