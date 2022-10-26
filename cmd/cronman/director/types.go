package director

import (
	"time"

	"github.com/tjper/rustcron/cmd/cronman/controller"
	"github.com/tjper/rustcron/cmd/cronman/lock"
	"github.com/tjper/rustcron/cmd/cronman/redis"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
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

type Director struct {
	logger     *zap.Logger
	redis      *redis.Redis
	controller *controller.Controller
	store      *gorm.DB

	distributedLock *lock.Distributed
}

func New(
	logger *zap.Logger,
	redis *redis.Redis,
	store *gorm.DB,
	controller *controller.Controller,
) *Director {
	return &Director{
		logger:          logger.With(zap.String("director-id", uuid.NewString())),
		redis:           redis,
		store:           store,
		controller:      controller,
		distributedLock: lock.NewDistributed(logger, redis, mutexKey, 2*time.Second),
	}
}
