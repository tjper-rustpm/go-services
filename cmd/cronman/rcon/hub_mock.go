package rcon

import (
	"context"
	"fmt"
)

// NewHubMock creates a HubMock instance.
func NewHubMock() *HubMock {
	return &HubMock{
		stack: make([]string, 0),
	}
}

// HubMock mocks a Hub instance for testing.
type HubMock struct {
	stack []string
}

// Dial mocks the dialing and creation of a IRcon instance.
func (h *HubMock) Dial(ctx context.Context, url, password string) (IRcon, error) {
	return &ClientMock{
		url:      url,
		password: password,
		hub:      h,
	}, nil
}

// LPop pops of the oldest item on the HubMock's internal stack.
func (h *HubMock) LPop() string {
	if len(h.stack) == 0 {
		return ""
	}
	item := h.stack[0]
	h.stack = h.stack[1:]
	return item
}

// ClientMock mocks Client.
type ClientMock struct {
	url      string
	password string
	hub      *HubMock
}

// Close mocks Client.Close.
func (m ClientMock) Close() {}

// Quit mocks Client.Quit.
func (m ClientMock) Quit(_ context.Context) error { return nil }

// AddModerator mocks Client.AddModerator.
func (m ClientMock) AddModerator(_ context.Context, _ string) error { return nil }

// RemoveModerator mocks Client.RemoveModerator.
func (m ClientMock) RemoveModerator(_ context.Context, _ string) error { return nil }

// GrantPermission mocks Client.GrantPermission.
func (m ClientMock) GrantPermission(_ context.Context, steamID string, permission string) error {
	m.hub.stack = append(m.hub.stack, fmt.Sprintf("%s %s %s %s", m.url, m.password, steamID, permission))
	return nil
}

// RevokePermission mocks Client.RevokePermission.
func (m ClientMock) RevokePermission(_ context.Context, _ string, _ string) error { return nil }

// CreateGroup mocks Client.CreateGroup.
func (m ClientMock) CreateGroup(_ context.Context, group string) error { return nil }

// AddToGroup mocks Client.AddToGroup.
func (m ClientMock) AddToGroup(_ context.Context, steamID string, group string) error {
	m.hub.stack = append(m.hub.stack, fmt.Sprintf("%s %s %s %s", m.url, m.password, steamID, group))
	return nil
}
