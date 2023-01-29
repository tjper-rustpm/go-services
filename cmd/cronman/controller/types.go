package controller

import (
	"context"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/cmd/cronman/server"
	itime "github.com/tjper/rustcron/internal/time"
	"gorm.io/gorm"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StreamWriter writes a slice of bytes to an event stream.
type StreamWriter interface {
	Write(context.Context, []byte) error
}

// IServerManager represents the API by which the Controller interacts with
// Rust servers.
type IServerManager interface {
	CreateInstance(ctx context.Context, template model.InstanceKind) (*server.CreateInstanceOutput, error)
	StartInstance(ctx context.Context, id string, userdata string) error
	StopInstance(ctx context.Context, id string) error
	MakeInstanceAvailable(ctx context.Context, instanceID, allocationID string) (*server.AssociationOutput, error)
	MakeInstanceUnavailable(ctx context.Context, associationID string) error
}

// ITime represents the API by which the cronman Controller interacts with
// time. See the corresponding definitions in the time package for more
// details.
type ITime interface {
	Now() time.Time
	Until(time.Time) time.Duration
}

// IHub represents the API by which IRcon types may be created.
type IHub interface {
	Dial(context.Context, string, string) (rcon.IRcon, error)
}

// IWaiter represents the API by which the Controller waits for Rcon endpoints
// to be ready.
type IWaiter interface {
	UntilReady(ctx context.Context, url string) error
}

// INotifier represents the API by which the Resolver notifies the Controller
// that the Rustpm datastore has changed.
type INotifier interface {
	Notify(ctx context.Context) error
}

// New creates a new Controller object.
func New(
	logger *zap.Logger,
	store *gorm.DB,
	serverController *ServerDirector,
	hub IHub,
	waiter IWaiter,
	notifier INotifier,
	eventStream StreamWriter,
) *Controller {
	return &Controller{
		logger:           logger.With(zap.String("controller-id", uuid.NewString())),
		time:             new(itime.Time),
		store:            store,
		serverController: serverController,
		hub:              hub,
		waiter:           waiter,
		notifier:         notifier,
		eventStream:      eventStream,
	}
}

// Controller is responsible for accumulating all server events, processing these
// events, and watching for event changes.
type Controller struct {
	logger *zap.Logger
	time   ITime

	store *gorm.DB

	// TODO: Should rename this to director or serverDirector.
	serverController *ServerDirector
	hub              IHub
	waiter           IWaiter
	notifier         INotifier
	eventStream      StreamWriter
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
