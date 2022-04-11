package controller

import "context"

// NewRconHubMock creates a RconHubMock instance.
func NewRconHubMock() *RconHubMock {
	return &RconHubMock{}
}

// RconHubMock mocks an API providing access to a cronman server's Rcon
// functionality.
type RconHubMock struct{}

// Dial mocks the dialing and creation of a IRcon instance.
func (h RconHubMock) Dial(ctx context.Context, url, password string) (IRcon, error) {
	return &RconMock{}, nil
}

// RconMock mocks an Rcon connection to a cronman server and the available
// functionality.
type RconMock struct{}

// Close mocks closing a Rcon connection.
func (m RconMock) Close() {}

// Quit mocks instructing the cronman server to quit via the Rcon interface.
func (m RconMock) Quit(_ context.Context) error { return nil }

// AddModerator mocks instructing the cronman server to add the specified
// moderator.
func (m RconMock) AddModerator(_ context.Context, _ string) error { return nil }

// RemoveModerator mocks instructing the cronman server to remove the specified
// moderator.
func (m RconMock) RemoveModerator(_ context.Context, _ string) error { return nil }
