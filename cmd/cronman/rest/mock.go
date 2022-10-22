package rest

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/cronman/controller"
	"github.com/tjper/rustcron/cmd/cronman/model"
)

// ErrMisconfiguredMock indicates an isntance of ControllerMock is being
// utilized in an unexpected way.
var ErrMisconfiguredMock = errors.New("mock configuration does not support this functionality")

// NewControllerMock creates a new ControllerMock instance. Utilize
// ControllerMockOption functions to configure the instance.
func NewControllerMock(options ...ControllerMockOption) *ControllerMock {
	mock := &ControllerMock{}

	for _, option := range options {
		option(mock)
	}
	return mock
}

// ControllerMockOption is a function type that should configure the
// ControllerMock instance.
type ControllerMockOption func(*ControllerMock)

// WithCreateServer provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock CreateServer
// functionality.
func WithCreateServer(fn createServerFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.createServer = fn
	}
}

// WithGetServer provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock GetServer
// functionality.
func WithGetServer(fn getServerFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.getServer = fn
	}
}

// WithUpdateServer provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock UpdateServer
// functionality.
func WithUpdateServer(fn updateServerFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.updateServer = fn
	}
}

// WithArchiveServer provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock ArchiveServer
// functionality.
func WithArchiveServer(fn archiveServerFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.archiveServer = fn
	}
}

// WithStartServer provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock StartServer
// functionality.
func WithStartServer(fn startServerFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.startServer = fn
	}
}

// WithMakeServerLive provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock MakeServerLive
// functionality.
func WithMakeServerLive(fn makeServerLiveFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.makeServerLive = fn
	}
}

// WithStopServer provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock StopServer
// functionality.
func WithStopServer(fn stopServerFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.stopServer = fn
	}
}

// WithWipeServer provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock WipeServer
// functionality.
func WithWipeServer(fn wipeServerFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.wipeServer = fn
	}
}

// WithListServers provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock ListServers
// functionality.
func WithListServers(fn listServersFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.listServers = fn
	}
}

// WithAddServerTags provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock AddServerTags
// functionality.
func WithAddServerTags(fn addServerTagsFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.addServerTags = fn
	}
}

// WithRemoveServerTags provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock RemoveServerTags
// functionality.
func WithRemoveServerTags(fn removeServerTagsFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.removeServerTags = fn
	}
}

// WithAddServerEvents provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock AddServerEvents
// functionality.
func WithAddServerEvents(fn addServerEventsFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.addServerEvents = fn
	}
}

// WithRemoveServerEvents provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock RemoveServerEvents
// functionality.
func WithRemoveServerEvents(fn removeServerEventsFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.removeServerEvents = fn
	}
}

// WithAddServerModerators provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock AddServerModerators
// functionality.
func WithAddServerModerators(fn addServerModeratorsFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.addServerModerators = fn
	}
}

// WithRemoveServerModerators provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock RemoveServerModerators
// functionality.
func WithRemoveServerModerators(fn removeServerModeratorsFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.removeServerModerators = fn
	}
}

// WithAddServerOwners provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock AddServerOwners
// functionality.
func WithAddServerOwners(fn addServerOwnersFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.addServerOwners = fn
	}
}

// WithRemoveServerOwners provides a ControllerMockOption that configures a
// ControllerMock to utilize the passed function to mock RemoveServerOwners
// functionality.
func WithRemoveServerOwners(fn removeServerOwnersFunc) ControllerMockOption {
	return func(mock *ControllerMock) {
		mock.removeServerOwners = fn
	}
}

type (
	createServerFunc           func(context.Context, model.Server) (*model.DormantServer, error)
	getServerFunc              func(context.Context, uuid.UUID) (interface{}, error)
	updateServerFunc           func(context.Context, controller.UpdateServerInput) (*model.DormantServer, error)
	archiveServerFunc          func(context.Context, uuid.UUID) (*model.ArchivedServer, error)
	startServerFunc            func(context.Context, uuid.UUID) (*model.DormantServer, error)
	makeServerLiveFunc         func(context.Context, uuid.UUID) (*model.LiveServer, error)
	stopServerFunc             func(context.Context, uuid.UUID) (*model.DormantServer, error)
	wipeServerFunc             func(context.Context, uuid.UUID, model.Wipe) error
	listServersFunc            func(context.Context, interface{}) error
	addServerTagsFunc          func(context.Context, uuid.UUID, model.Tags) error
	removeServerTagsFunc       func(context.Context, uuid.UUID, []uuid.UUID) error
	addServerEventsFunc        func(context.Context, uuid.UUID, model.Events) error
	removeServerEventsFunc     func(context.Context, uuid.UUID, []uuid.UUID) error
	addServerModeratorsFunc    func(context.Context, uuid.UUID, model.Moderators) error
	removeServerModeratorsFunc func(context.Context, uuid.UUID, []uuid.UUID) error
	addServerOwnersFunc        func(context.Context, uuid.UUID, model.Owners) error
	removeServerOwnersFunc     func(context.Context, uuid.UUID, []uuid.UUID) error
)

// ControllerMock is typically used to implement the IController interface for
// testing purposes.
type ControllerMock struct {
	createServer           createServerFunc
	getServer              getServerFunc
	updateServer           updateServerFunc
	archiveServer          archiveServerFunc
	startServer            startServerFunc
	makeServerLive         makeServerLiveFunc
	stopServer             stopServerFunc
	wipeServer             wipeServerFunc
	listServers            listServersFunc
	addServerTags          addServerTagsFunc
	removeServerTags       removeServerTagsFunc
	addServerEvents        addServerEventsFunc
	removeServerEvents     removeServerEventsFunc
	addServerModerators    addServerModeratorsFunc
	removeServerModerators removeServerModeratorsFunc
	addServerOwners        addServerOwnersFunc
	removeServerOwners     removeServerOwnersFunc
}

// CreateServer executes the handler set with WithCreateServer.
func (m ControllerMock) CreateServer(ctx context.Context, server model.Server) (*model.DormantServer, error) {
	if m.createServer == nil {
		return nil, ErrMisconfiguredMock
	}
	return m.createServer(ctx, server)
}

// GetServer executes the handler set with WithGetServer.
func (m ControllerMock) GetServer(ctx context.Context, id uuid.UUID) (interface{}, error) {
	if m.getServer == nil {
		return nil, ErrMisconfiguredMock
	}
	return m.getServer(ctx, id)
}

// UpdateServer executes the handler set with WithUpdateServer.
func (m ControllerMock) UpdateServer(ctx context.Context, input controller.UpdateServerInput) (*model.DormantServer, error) {
	if m.updateServer == nil {
		return nil, ErrMisconfiguredMock
	}
	return m.updateServer(ctx, input)
}

// ArchiveServer executes the handler set with WithArchiveServer.
func (m ControllerMock) ArchiveServer(ctx context.Context, id uuid.UUID) (*model.ArchivedServer, error) {
	if m.archiveServer == nil {
		return nil, ErrMisconfiguredMock
	}
	return m.archiveServer(ctx, id)
}

// StartServer executes the handler set with WithStartServer.
func (m ControllerMock) StartServer(ctx context.Context, id uuid.UUID) (*model.DormantServer, error) {
	if m.startServer == nil {
		return nil, ErrMisconfiguredMock
	}
	return m.startServer(ctx, id)
}

// MakeServerLive executes the handler set with WithMakeServerLive.
func (m ControllerMock) MakeServerLive(ctx context.Context, id uuid.UUID) (*model.LiveServer, error) {
	if m.makeServerLive == nil {
		return nil, ErrMisconfiguredMock
	}
	return m.makeServerLive(ctx, id)
}

// StopServer executes the handler set with WithStopServer.
func (m ControllerMock) StopServer(ctx context.Context, id uuid.UUID) (*model.DormantServer, error) {
	if m.stopServer == nil {
		return nil, ErrMisconfiguredMock
	}
	return m.stopServer(ctx, id)
}

// WipeServer executes the handler set with WithWipeServer.
func (m ControllerMock) WipeServer(ctx context.Context, id uuid.UUID, wipe model.Wipe) error {
	if m.wipeServer == nil {
		return ErrMisconfiguredMock
	}
	return m.wipeServer(ctx, id, wipe)
}

// ListServers executes the handler set with WithListServers.
func (m ControllerMock) ListServers(ctx context.Context, dest interface{}) error {
	if m.listServers == nil {
		return ErrMisconfiguredMock
	}
	return m.listServers(ctx, dest)
}

// AddServerTags executes the handler set with WithAddServerTags.
func (m ControllerMock) AddServerTags(ctx context.Context, id uuid.UUID, tags model.Tags) error {
	if m.addServerTags == nil {
		return ErrMisconfiguredMock
	}
	return m.addServerTags(ctx, id, tags)
}

// RemoveServerTags executes the handler set with WithRemoveServerTags.
func (m ControllerMock) RemoveServerTags(ctx context.Context, id uuid.UUID, ids []uuid.UUID) error {
	if m.removeServerTags == nil {
		return ErrMisconfiguredMock
	}
	return m.removeServerTags(ctx, id, ids)
}

// AddServerEvents executes the handler set with WithAddServerEvents.
func (m ControllerMock) AddServerEvents(ctx context.Context, id uuid.UUID, events model.Events) error {
	if m.addServerEvents == nil {
		return ErrMisconfiguredMock
	}
	return m.addServerEvents(ctx, id, events)
}

// RemoveServerEvents executes the handler set with WithRemoveServerEvents.
func (m ControllerMock) RemoveServerEvents(ctx context.Context, id uuid.UUID, ids []uuid.UUID) error {
	if m.removeServerEvents == nil {
		return ErrMisconfiguredMock
	}
	return m.removeServerEvents(ctx, id, ids)
}

// AddServerModerators executes the handler set with WithAddServerModerators.
func (m ControllerMock) AddServerModerators(ctx context.Context, id uuid.UUID, moderators model.Moderators) error {
	if m.addServerModerators == nil {
		return ErrMisconfiguredMock
	}
	return m.addServerModerators(ctx, id, moderators)
}

// RemoveServerModerators executes the handler set with WithRemoveServerModerators.
func (m ControllerMock) RemoveServerModerators(ctx context.Context, id uuid.UUID, ids []uuid.UUID) error {
	if m.removeServerModerators == nil {
		return ErrMisconfiguredMock
	}
	return m.removeServerModerators(ctx, id, ids)
}

// AddServerOwners executes the handler set with WithAddServerOwners.
func (m ControllerMock) AddServerOwners(ctx context.Context, id uuid.UUID, owners model.Owners) error {
	if m.addServerOwners == nil {
		return ErrMisconfiguredMock
	}
	return m.addServerOwners(ctx, id, owners)
}

// RemoveServerOwners executes the handler set with WithRemoveServerOwners.
func (m ControllerMock) RemoveServerOwners(ctx context.Context, id uuid.UUID, ids []uuid.UUID) error {
	if m.removeServerOwners == nil {
		return ErrMisconfiguredMock
	}
	return m.removeServerOwners(ctx, id, ids)
}
