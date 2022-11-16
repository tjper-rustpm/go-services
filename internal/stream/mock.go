package stream

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

// WithWrite returns a ClientMockOption that configures a ClientMock to call fn
// when Write is called.
func WithWrite(fn writeFunc) ClientMockOption {
	return func(mock *ClientMock) { mock.write = fn }
}

// WithClaim returns a ClientMockOption that configures a ClientMock to call fn
// when Claim is called.
func WithClaim(fn claimFunc) ClientMockOption {
	return func(mock *ClientMock) { mock.claim = fn }
}

// WithRead returns a ClientMockOption that configures a ClientMock to call fn
// when Read is called.
func WithRead(fn readFunc) ClientMockOption {
	return func(mock *ClientMock) { mock.read = fn }
}

// WithAck returns a ClientMockOption that configures a ClientMock to call fn
// when Ack is called.
func WithAck(fn ackFunc) ClientMockOption {
	return func(mock *ClientMock) { mock.ack = fn }
}

type (
	writeFunc func(context.Context, []byte) error
	claimFunc func(context.Context, time.Duration) (*Message, error)
	readFunc  func(context.Context) (*Message, error)
	ackFunc   func(context.Context, *Message) error
)

// ClientMock provides an implementation for mock stream.Client interactions.
// This is typically used for unit-testing.
type ClientMock struct {
	write writeFunc
	claim claimFunc
	read  readFunc
	ack   ackFunc
}

// Write calls the function configured with WithWrite.
func (mock ClientMock) Write(ctx context.Context, b []byte) error {
	if mock.write == nil {
		return errUnconfigured
	}
	return mock.write(ctx, b)
}

// Claim calls the function configured with WithClaim.
func (mock ClientMock) Claim(ctx context.Context, idle time.Duration) (*Message, error) {
	if mock.claim == nil {
		return nil, errUnconfigured
	}
	return mock.claim(ctx, idle)
}

// Read calls the function configured with WithRead.
func (mock ClientMock) Read(ctx context.Context) (*Message, error) {
	if mock.read == nil {
		return nil, errUnconfigured
	}
	return mock.read(ctx)
}

// Ack calls the function configured with WithAck.
func (mock ClientMock) Ack(ctx context.Context, m *Message) error {
	if mock.ack == nil {
		return errUnconfigured
	}
	return mock.ack(ctx, m)
}
