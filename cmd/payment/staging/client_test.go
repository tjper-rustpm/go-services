// +build integration

package staging

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	redisAddr = flag.String(
		"redis-addr",
		"redis:6379",
		"address of redis instance to be used for integration testing",
	)
	redisPassword = flag.String(
		"redis-password",
		"",
		"password to access redis instance to be used for integration testing",
	)
)

func TestIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	expected := Checkout{
		ServerID: uuid.New(),
		UserID:   uuid.New(),
	}

	var id string
	t.Run("stage checkout", func(t *testing.T) {
		res, err := suite.client.StageCheckout(ctx, expected, time.Now().Add(2*time.Second))
		assert.Nil(t, err)
		id = res
	})

	t.Run("fetch checkout", func(t *testing.T) {
		actual, err := suite.client.FetchCheckout(ctx, id)
		assert.Nil(t, err)
		assert.Equal(t, expected, *actual)
	})

	time.Sleep(3 * time.Second)
	t.Run("fetch expired checkout", func(t *testing.T) {
		_, err := suite.client.FetchCheckout(ctx, id)
		assert.Error(t, err)
	})
}

func setup(ctx context.Context, t *testing.T) *suite {
	redis := redis.NewClient(
		&redis.Options{
			Addr:     *redisAddr,
			Password: *redisPassword,
		},
	)
	err := redis.Ping(ctx).Err()
	require.Nil(t, err)

	return &suite{
		client: NewClient(redis),
	}
}

type suite struct {
	client *Client
}
