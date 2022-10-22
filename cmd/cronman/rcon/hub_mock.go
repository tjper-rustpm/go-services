package rcon

import (
	"context"
	"fmt"
)

// NewHubMock creates a HubMock instance.
func NewHubMock(options ...HubMockOption) *HubMock {
	m := &HubMock{
		stack: make([]string, 0),
	}

	for _, option := range options {
		option(m)
	}
	return m
}

// HubMockOption mutates a HubMock instance. Typically used with NewHubMock
// to allow configure a HubMock instance.
type HubMockOption func(*HubMock)

// WithServerInfo is a HubMockOption that configures the HubMock's ServerInfo
// response used for ClientMock.ServerInfo calls.
func WithServerInfo(serverInfo ServerInfo) HubMockOption {
	return func(m *HubMock) {
		m.serverInfo = serverInfo
	}
}

// HubMock mocks a Hub instance for testing.
type HubMock struct {
	stack      []string
	serverInfo ServerInfo
}

// Dial mocks the dialing and creation of a IRcon instance.
func (h *HubMock) Dial(ctx context.Context, url, password string) (IRcon, error) {
	return &ClientMock{
		url:      url,
		password: password,
		hub:      h,
		msgc:     make(chan string, 128),
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

	hub  *HubMock
	msgc chan string
}

// Close mocks Client.Close.
func (m ClientMock) Close() {}

// Quit mocks Client.Quit.
func (m ClientMock) Quit(_ context.Context) error { return nil }

// Say mocks Client.Say.
func (m ClientMock) Say(_ context.Context, msg string) error {
	m.msgc <- msg
	return nil
}

// Said retrieves the most recent message written to the ClientMock via Say.
func (m ClientMock) Said(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case msg := <-m.msgc:
		return msg, nil
	}
}

// AddModerator mocks Client.AddModerator.
func (m ClientMock) AddModerator(_ context.Context, _ string) error { return nil }

// RemoveModerator mocks Client.RemoveModerator.
func (m ClientMock) RemoveModerator(_ context.Context, _ string) error { return nil }

// AddOwner mocks Client.AddOwner.
func (m ClientMock) AddOwner(_ context.Context, _ string) error { return nil }

// RemoveOwner mocks Client.RemoveOwner.
func (m ClientMock) RemoveOwner(_ context.Context, _ string) error { return nil }

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

// ServerInfo mocks Client.ServerInfo.
func (m ClientMock) ServerInfo(_ context.Context) (*ServerInfo, error) {
	return &m.hub.serverInfo, nil
}
