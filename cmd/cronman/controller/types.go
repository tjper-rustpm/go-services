package controller

import (
	"context"

	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/cmd/cronman/server"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// IServerManager represents the API by which the Controller interacts with
// Rust servers.
type IServerManager interface {
	CreateInstance(ctx context.Context, template model.InstanceKind) (*server.CreateInstanceOutput, error)
	StartInstance(ctx context.Context, id string, userdata string) error
	StopInstance(ctx context.Context, id string) error
	MakeInstanceAvailable(ctx context.Context, instanceId, allocationId string) (*server.AssociationOutput, error)
	MakeInstanceUnavailable(ctx context.Context, associationId string) error
}

// IHub represents the API by which IRcon types may be created.
type IHub interface {
	Dial(context.Context, string, string) (IRcon, error)
}

// IWaiter represents the API by which the Controller waits for Rcon endpoints
// to be ready.
type IWaiter interface {
	UntilReady(ctx context.Context, url string) error
}

// IRcon represents the API by which the Controller communicates and interacts
// with its Rust server rcons.
type IRcon interface {
	Close()
	Quit(ctx context.Context) error
	AddModerator(ctx context.Context, id string) error
	RemoveModerator(ctx context.Context, id string) error
}

// INotifier represents the API by which the Resolver notifies the Controller
// that the Rustpm datastore has changed.
type INotifier interface {
	Notify(ctx context.Context) error
}

// New creates a new Controller object.
func New(
	logger *zap.Logger,
	store db.IStore,
	serverController *ServerDirector,
	hub IHub,
	waiter IWaiter,
	notifier INotifier,
) *Controller {
	return &Controller{
		logger:           logger.With(zap.String("controller-id", uuid.NewString())),
		store:            store,
		serverController: serverController,
		hub:              hub,
		waiter:           waiter,
		notifier:         notifier,
	}
}

// Controller is responsible for accumulating all server events, processing these
// events, and watching for event changes.
type Controller struct {
	logger *zap.Logger

	store            db.IStore
	serverController *ServerDirector
	hub              IHub
	waiter           IWaiter
	notifier         INotifier
}

// NewServerDirerctor creates a new ServerDirector object.
func NewServerDirector(usEast, usWest, euCentral IServerManager) *ServerDirector {
	return &ServerDirector{
		managers: map[model.Region]IServerManager{
			model.RegionUsEast:    usEast,
			model.RegionUsWest:    usWest,
			model.RegionEuCentral: euCentral,
		},
	}
}

// ServerDirector is responsible for exposing the server Managers for use.
type ServerDirector struct {
	managers map[model.Region]IServerManager
}

// Region retrieves the Manager allocated to the specified region.
func (dir ServerDirector) Region(region model.Region) IServerManager {
	return dir.managers[region]
}
