package server

import (
	"context"

	"github.com/tjper/rustcron/cmd/cronman/model"
)

// NewMockManager creates a MockManager instance.
func NewMockManager() *MockManager {
	return &MockManager{}
}

// MockManager provides methods to mock interactions with cronman servers. This
// is typically used in testing to avoid interacting with AWS.
type MockManager struct {
	createInstanceHandler        func(context.Context, model.InstanceKind) (*CreateInstanceOutput, error)
	makeInstanceAvailableHandler func(context.Context, string, string) (*AssociationOutput, error)
	startInstanceHandler         func(context.Context, string, string) error
}

// SetCreateInstanceHandler sets the handler of the CreateInstance method to
// the passed function.
func (m *MockManager) SetCreateInstanceHandler(handler func(context.Context, model.InstanceKind) (*CreateInstanceOutput, error)) {
	m.createInstanceHandler = handler
}

// CreateInstance mocks the creation of a cronman server instance.
func (m MockManager) CreateInstance(ctx context.Context, kind model.InstanceKind) (*CreateInstanceOutput, error) {
	if m.createInstanceHandler == nil {
		return nil, nil
	}
	return m.createInstanceHandler(ctx, kind)
}

// SetStartInstanceHandler sets the handler of the StartInstance method to the
// passed function.
func (m *MockManager) SetStartInstanceHandler(handler func(context.Context, string, string) error) {
	m.startInstanceHandler = handler
}

// StartInstance mocks the starting of a cronman server instance.
func (m MockManager) StartInstance(ctx context.Context, id string, userdata string) error {
	if m.startInstanceHandler == nil {
		return nil
	}
	return m.startInstanceHandler(ctx, id, userdata)
}

// StopInstance mocks the stopping of a cronman server instance.
func (m MockManager) StopInstance(_ context.Context, _ string) error {
	return nil
}

// SetMakeInstanceAvailableHandler sets the handler of the MakeInstanceAvailable
// method to the passed function.
func (m *MockManager) SetMakeInstanceAvailableHandler(handler func(context.Context, string, string) (*AssociationOutput, error)) {
	m.makeInstanceAvailableHandler = handler
}

// MakeInstanceAvailable mocks the making a cronman server instance available.
func (m MockManager) MakeInstanceAvailable(ctx context.Context, instanceID string, allocationID string) (*AssociationOutput, error) {
	if m.makeInstanceAvailableHandler == nil {
		return nil, nil
	}
	return m.makeInstanceAvailableHandler(ctx, instanceID, allocationID)
}

// MakeInstanceUnavailable mocks the making a cronman server instance
// unavailable.
func (m MockManager) MakeInstanceUnavailable(_ context.Context, _ string) error {
	return nil
}
