package stream

import (
	"context"
	"testing"
	"time"

	"github.com/tjper/rustcron/internal/event"

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

	client, err := Init(ctx, rdb, "test-suite")
	require.Nil(t, err)

	return &Suite{Client: client}
}

type Suite struct {
	Client *Client
}

func (s Suite) ReadEvent(
	ctx context.Context,
	t *testing.T,
) interface{} {
	t.Helper()

	m, err := s.Client.Read(ctx)
	require.Nil(t, err)

	eventI, err := event.Parse(m.Payload)
	require.Nil(t, err)

	err = m.Ack(ctx)
	require.Nil(t, err)

	return eventI
}

func (s Suite) AssertNoEvent(ctx context.Context, t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err := s.Client.Read(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
