// +build integration

package stream

import (
	"context"
	"errors"
	"flag"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
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

func TestRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	t.Run("read while stream empty", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		_, err := suite.stream.Read(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("write", func(t *testing.T) {
		err := suite.stream.Write(ctx, []byte("message"))
		assert.Nil(t, err)
	})

	t.Run("read", func(t *testing.T) {
		m, err := suite.stream.Read(ctx)
		assert.Nil(t, err)
		assert.Equal(t, []byte("message"), m.Payload)

		err = m.Ack(ctx)
		assert.Nil(t, err)
	})
}

func TestMultipleReadersAndWriters(t *testing.T) {
	tests := map[string]struct {
		timeout  time.Duration
		writers  int
		readers  int
		messages int
	}{
		"1 writer - 1 readers":   {timeout: 2 * time.Second, writers: 1, readers: 1, messages: 1000},
		"1 writer - 2 readers":   {timeout: 2 * time.Second, writers: 1, readers: 2, messages: 1000},
		"2 writer - 1 readers":   {timeout: 2 * time.Second, writers: 2, readers: 1, messages: 1000},
		"2 writer - 2 readers":   {timeout: 2 * time.Second, writers: 2, readers: 2, messages: 1000},
		"10 writer - 10 readers": {timeout: 2 * time.Second, writers: 4, readers: 4, messages: 10000},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), test.timeout)
			defer cancel()

			suite := setup(ctx, t)

			sendc := make(chan int, test.messages)
			receivec := make(chan []byte, test.messages)

			for i := 0; i < test.writers; i++ {
				go func(i int) {
					for msg := range sendc {
						err := suite.stream.Write(ctx, []byte(strconv.Itoa(msg)))
						assert.Nil(t, err)
					}
				}(i)
			}

			for i := 0; i < test.readers; i++ {
				go func(i int) {
					for {
						m, err := suite.stream.Read(ctx)
						if errors.Is(err, context.DeadlineExceeded) {
							return
						}

						assert.Nil(t, err)
						receivec <- m.Payload

						err = m.Ack(ctx)
						assert.Nil(t, err)
					}
				}(i)
			}

			for i := 0; i < test.messages; i++ {
				sendc <- i
			}
			close(sendc)

			received := make([]int, 0)
			for {
				select {
				case <-ctx.Done():
					assert.Equal(t, test.messages, len(received))
					return
				case msg := <-receivec:
					i, err := strconv.Atoi(string(msg))
					assert.Nil(t, err)

					received = append(received, i)
				}
			}
		})
	}
}

func TestFatalRecovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	alpha := setup(ctx, t)
	bravo := setup(ctx, t)

	t.Run("alpha write", func(t *testing.T) {
		err := alpha.stream.Write(ctx, []byte("message"))
		assert.Nil(t, err)
	})

	t.Run("alpha read w/ no ack", func(t *testing.T) {
		m, err := alpha.stream.Read(ctx)
		assert.Nil(t, err)
		assert.Equal(t, []byte("message"), m.Payload)
	})

	t.Run("bravo read", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		_, err := bravo.stream.Read(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("bravo claim", func(t *testing.T) {
		m, err := bravo.stream.Claim(ctx, time.Second)
		assert.Nil(t, err)
		assert.Equal(t, []byte("message"), m.Payload)

		err = m.Ack(ctx)
		assert.Nil(t, err)
	})

	t.Run("alpha claim w/ empty stream", func(t *testing.T) {
		_, err := alpha.stream.Claim(ctx, time.Second)
		assert.ErrorIs(t, err, ErrNoPending)
	})
}

func setup(ctx context.Context, t *testing.T) *suite {
	rdb := redis.NewClient(
		&redis.Options{
			Addr:     *redisAddr,
			Password: *redisPassword,
		},
	)
	err := rdb.Ping(ctx).Err()
	require.Nil(t, err)

	err = rdb.FlushDB(ctx).Err()
	require.Nil(t, err)

	stream, err := Init(ctx, rdb, "test")
	require.Nil(t, err)

	return &suite{
		rdb:    rdb,
		stream: stream,
	}
}

type suite struct {
	rdb    *redis.Client
	stream *Client
}
