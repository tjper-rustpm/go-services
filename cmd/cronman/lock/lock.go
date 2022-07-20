package lock

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// IRedis is represents the API by which the Redis can be communicated with.
type IRedis interface {
	SetNX(context.Context, string, interface{}, time.Duration) (bool, error)
	SetXX(context.Context, string, interface{}, time.Duration) (bool, error)
}

// NewDistributed creates a Distributed instance. The key is the redis key the
// lock will use. All Distributed instances with the same redis instance and
// key will contend for the same lock. The expiration is the rate at which the
// lock is refreshed.
func NewDistributed(
	logger *zap.Logger,
	redis IRedis,
	key string,
	expiration time.Duration,
) *Distributed {
	return &Distributed{
		logger:     logger,
		redis:      redis,
		key:        key,
		expiration: expiration,
	}
}

// Distributed is a distributed lock that utilizes Redis.
// IMPORTANT: This distributed lock is only effective on single instance
// of Redis. If a Redis cluster is being used, look into Redigo.
type Distributed struct {
	logger *zap.Logger
	redis  IRedis

	expiration time.Duration
	key        string
	cancel     context.CancelFunc
}

// Lock seeks to acquire the distributed lock. This method blocks until this
// distributed lock is acquired or the context is cancelled.
func (d *Distributed) Lock(ctx context.Context) error {
	ticker := time.NewTicker(d.expiration / 2)
retry:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			set, err := d.redis.SetNX(ctx, d.key, time.Now(), d.expiration)
			if err != nil {
				return err
			}
			if set {
				d.logger.Info("lock acquired")
				break retry
			}
		}
	}

	// launch goroutine that periodically refreshes the distributed lock once it
	// has been acquired. As long at the distributed lock is being refreshed, the
	// application that originally acquired the distributed lock will keep it.
	lockCtx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	go func() {
		defer cancel()
		d.maintainLock(lockCtx)
	}()
	return nil
}

// Unlock releases the distributed lock.
func (d *Distributed) Unlock(ctx context.Context) {
	d.cancel()
}

// --- private ---

// maintainLock maintains the lock once it has been acquired. As long at the
// distributed lock is being refreshed, the application that originally
// acquired the distributed lock will keep it.
func (d *Distributed) maintainLock(ctx context.Context) {
	ticker := time.NewTicker(d.expiration / 2)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			set, err := d.redis.SetXX(ctx, d.key, time.Now(), d.expiration)
			if err != nil {
				d.logger.Fatal("failed to maintain distributed lock", zap.Error(err))
			}
			if !set {
				d.logger.Fatal("failed to maintain distributed lock; set failed")
			}
		}
	}
}
