package rcon

import (
	"context"
	"time"
)

// NewWaiterMock creates a WaiterMock instance.
func NewWaiterMock(until time.Duration) *WaiterMock {
	return &WaiterMock{readyAt: time.Now().Add(until)}
}

// WaiterMock mocks Waiter functionality. Typically used in testing.
type WaiterMock struct {
	readyAt time.Time
}

// UntilReady mocks waiting until the specified URL is accepting websocket
// connections.
func (m WaiterMock) UntilReady(ctx context.Context, _ string, wait time.Duration) error {
	ticker := time.NewTicker(wait)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(m.readyAt) {
				return nil
			}
		}
	}
}
