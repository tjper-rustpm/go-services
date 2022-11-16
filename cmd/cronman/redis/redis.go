package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

func New(redis *redis.Client) *Redis {
	return &Redis{
		redis: redis,
	}
}

// Redis wraps the redis.Client. This is done to make the redis.Client
// simpler to test against.
type Redis struct {
	redis *redis.Client
}

// SetNX wraps redis.Client.SetNX.Result().
func (r Redis) SetNX(
	ctx context.Context,
	key string,
	val interface{},
	exp time.Duration,
) (bool, error) {
	return r.redis.SetNX(ctx, key, val, exp).Result()
}

// SetXX wraps redis.Client.SetXX.Result().
func (r Redis) SetXX(
	ctx context.Context,
	key string,
	val interface{},
	exp time.Duration,
) (bool, error) {
	return r.redis.SetXX(ctx, key, val, exp).Result()
}

// Subscribe wraps redis.Client.Subscribe.
func (r Redis) Subscribe(ctx context.Context, subject string) *redis.PubSub {
	return r.redis.Subscribe(ctx, subject)
}
