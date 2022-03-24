package redis

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func InitSuite(ctx context.Context, t *testing.T) *Suite {
	t.Helper()

	const (
		redisAddr     = "redis:6379"
		redisPassword = ""
	)

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
	})
	err := rdb.Ping(ctx).Err()
	require.Nil(t, err)

	return &Suite{Redis: rdb}
}

type Suite struct {
	Redis *redis.Client
}
