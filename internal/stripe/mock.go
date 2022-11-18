package stripe

import (
	"errors"

	"github.com/stripe/stripe-go/v72"
)

func NewMock(options ...MockOption) *Mock {
	mock := &Mock{}

	for _, option := range options {
		option(mock)
	}

	return mock
}

// MockOption is a function type that may configure a Mock instance.
type MockOption func(*Mock)

// WithCheckoutSession configures a Mock instance to execute the passed
// function when CheckoutSession is called.
func WithCheckoutSession(fn checkoutSessionFunc) MockOption {
	return func(mock *Mock) { mock.checkoutSession = fn }
}

// WithBillingPortalSession configures a Mock instance to execute the passed
// function when BillingPortalSession is called.
func WithBillingPortalSession(fn billingPortalSessionFunc) MockOption {
	return func(mock *Mock) { mock.billingPortalSession = fn }
}

// WithConstructEvent configures a Mock instance to execute the passed
// function when ConstructEvent is called.
func WithConstructEvent(fn constructEventFunc) MockOption {
	return func(mock *Mock) { mock.constructEvent = fn }
}

type (
	checkoutSessionFunc      func(*stripe.CheckoutSessionParams) (string, error)
	billingPortalSessionFunc func(*stripe.BillingPortalSessionParams) (string, error)
	constructEventFunc       func([]byte, string) (stripe.Event, error)
)

// Mock is responsible for mocking Stripe interactions this type is typically
// used during unit-testing.
type Mock struct {
	checkoutSession      checkoutSessionFunc
	billingPortalSession billingPortalSessionFunc
	constructEvent       constructEventFunc
}

// CheckoutSession calls the function configured via WithCheckoutSession.
func (m Mock) CheckoutSession(params *stripe.CheckoutSessionParams) (string, error) {
	if m.checkoutSession == nil {
		return "", errUnconfigured
	}
	return m.checkoutSession(params)
}

// BillingPortalSession calls the function configured via WithBillingPortalSession.
func (m *Mock) BillingPortalSession(params *stripe.BillingPortalSessionParams) (string, error) {
	if m.billingPortalSession == nil {
		return "", errUnconfigured
	}
	return m.billingPortalSession(params)
}

// ConstructEvent calls the function configured via WithConstructEvent.
func (m *Mock) ConstructEvent(b []byte, signature string) (stripe.Event, error) {
	if m.constructEvent == nil {
		return stripe.Event{}, errUnconfigured
	}
	return m.constructEvent(b, signature)
}

// errUnconfigured indicates a call was made that the mock was not correctly
// configured for.
var errUnconfigured = errors.New("unconfigured mock call")
