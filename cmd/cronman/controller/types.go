package controller

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/cronman/db"
	graphmodel "github.com/tjper/rustcron/cmd/cronman/graph/model"
	"github.com/tjper/rustcron/cmd/cronman/lock"
	"github.com/tjper/rustcron/cmd/cronman/redis"
	"github.com/tjper/rustcron/cmd/cronman/server"
	"go.uber.org/zap"
)

const (
	// This key is used to acquire a distributed Redis lock.
	mutexKey = "directing-lock-key"

	// This subject is used to notify the active controller that the server events
	// need to be re-evaluated. In a horizontally distributed system, it is not
	// known which instance will be directing the servers. Therefore, the
	// controller.Refresh method, must publish to this subject while a the
	// acting controller listens.
	refreshSubj = "controller-refresh"
)

// IServerManager represents the API by which the Controller interacts with
// Rust servers.
type IServerManager interface {
	CreateInstance(ctx context.Context, template graphmodel.InstanceKind) (*server.CreateInstanceOutput, error)
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
	UntilReady(ctx context.Context, url string, wait time.Duration) error
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
	redis *redis.Redis,
	store db.IStore,
	serverController *ServerDirector,
	hub IHub,
	waiter IWaiter,
	notifier INotifier,
) *Controller {
	return &Controller{
		logger:           logger.With(zap.String("controller-id", uuid.NewString())),
		redis:            redis,
		store:            store,
		serverController: serverController,
		hub:              hub,
		waiter:           waiter,
		notifier:         notifier,
		distributedLock:  lock.NewDistributed(logger, redis, mutexKey, 2*time.Second),
	}
}

// Controller is responsible for accumulating all server events, processing these
// events, and watching for event changes.
type Controller struct {
	logger *zap.Logger
	redis  *redis.Redis

	store            db.IStore
	serverController *ServerDirector
	hub              IHub
	waiter           IWaiter
	notifier         INotifier

	distributedLock *lock.Distributed
}

// NewServerDirerctor creates a new ServerDirector object.
func NewServerDirector(usEast, usWest, euCentral IServerManager) *ServerDirector {
	return &ServerDirector{
		managers: map[graphmodel.Region]IServerManager{
			graphmodel.RegionUsEast:    usEast,
			graphmodel.RegionUsWest:    usWest,
			graphmodel.RegionEuCentral: euCentral,
		},
	}
}

// ServerDirector is responsible for exposing the server Managers for use.
type ServerDirector struct {
	managers map[graphmodel.Region]IServerManager
}

// Region retrieves the Manager allocated to the specified region.
func (dir ServerDirector) Region(region graphmodel.Region) IServerManager {
	return dir.managers[region]
}
