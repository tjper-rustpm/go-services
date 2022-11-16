package staging

import (
	"context"
	"errors"
	"time"
)

var errUnconfigured = errors.New("unconfigured mock call")

// NewClientMock creates a new ClientMock instance.
func NewClientMock(options ...ClientMockOption) *ClientMock {
	mock := &ClientMock{}

	for _, option := range options {
		option(mock)
	}

	return mock
}

// ClientMockOption is a function type that may configure a ClientMock
// instance.
type ClientMockOption func(*ClientMock)

// WithStageCheckout returns a ClientMockOption that configures a ClientMock to
// call fn when StageCheckout is called.
func WithStageCheckout(fn stageCheckoutFunc) ClientMockOption {
	return func(mock *ClientMock) { mock.stageCheckout = fn }
}

// WithFetchCheckout returns a ClientMockOption that configures a ClientMock to
// call fn when FetchCheckout is called.
func WithFetchCheckout(fn fetchCheckoutFunc) ClientMockOption {
	return func(mock *ClientMock) { mock.fetchCheckout = fn }
}

type (
	stageCheckoutFunc func(context.Context, interface{}, time.Time) (string, error)
	fetchCheckoutFunc func(context.Context, string) (interface{}, error)
)

// ClientMock provides an implementation for mocking staging.Client
// interactions. This is typically utilized for unit-testing.
type ClientMock struct {
	stageCheckout stageCheckoutFunc
	fetchCheckout fetchCheckoutFunc
}

// StageCheckout calls the function configured with WithStageCheckout.
func (mock ClientMock) StageCheckout(ctx context.Context, checkout interface{}, expiresAt time.Time) (string, error) {
	if mock.stageCheckout == nil {
		return "", errUnconfigured
	}
	return mock.stageCheckout(ctx, checkout, expiresAt)
}

// FetchCheckout calls the function configured with WithFetchCheckout.
func (mock ClientMock) FetchCheckout(ctx context.Context, id string) (interface{}, error) {
	if mock.fetchCheckout == nil {
		return nil, errUnconfigured
	}
	return mock.fetchCheckout(ctx, id)
}
