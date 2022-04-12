package rcon

import (
	"context"
	"time"
)

// NewWaiterMock creates a WaiterMock instance.
func NewWaiterMock(until time.Duration) *WaiterMock {
	return &WaiterMock{until: until}
}

// WaiterMock mocks Waiter functionality. Typically used in testing.
type WaiterMock struct {
	until time.Duration
}

// UntilReady mocks waiting until the specified URL is accepting websocket
// connections.
func (m WaiterMock) UntilReady(ctx context.Context, _ string) error {
	ready := time.After(m.until)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ready:
			return nil
		}
	}
}
