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
	createInstanceOutput *CreateInstanceOutput
	createInstanceError  error

	makeInstanceAvailableOutput *AssociationOutput
	makeInstanceAvailableError  error

	startInstanceHandler func(context.Context, string, string) error
}

// SetCreateInstanceOutput sets the output of the instance's CreateInstance
// method.
func (m *MockManager) SetCreateInstanceOutput(v *CreateInstanceOutput, err error) {
	m.createInstanceOutput = v
	m.createInstanceError = err
}

// CreateInstance mocks the creation of a cronman server instance.
func (m MockManager) CreateInstance(_ context.Context, _ model.InstanceKind) (*CreateInstanceOutput, error) {
	return m.createInstanceOutput, m.createInstanceError
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

// SetMakeInstanceAvailableOutput sets the output of the instance's
// MakeInstanceAvailable method.
func (m *MockManager) SetMakeInstanceAvailableOutput(v *AssociationOutput, err error) {
	m.makeInstanceAvailableOutput = v
	m.makeInstanceAvailableError = err
}

// MakeInstanceAvailable mocks the making a cronman server instance available.
func (m MockManager) MakeInstanceAvailable(_ context.Context, _ string, _ string) (*AssociationOutput, error) {
	return m.makeInstanceAvailableOutput, m.makeInstanceAvailableError
}

// MakeInstanceUnavailable mocks the making a cronman server instance
// unavailable.
func (m MockManager) MakeInstanceUnavailable(_ context.Context, _ string) error {
	return nil
}
