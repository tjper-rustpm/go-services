//go:build integration
// +build integration

package stream

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/tjper/rustcron/internal/redis"

	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	t.Run("read while stream empty", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		_, err := suite.Client.Read(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("write", func(t *testing.T) {
		err := suite.Client.Write(ctx, []byte("message"))
		require.Nil(t, err)
	})

	t.Run("read", func(t *testing.T) {
		m, err := suite.Client.Read(ctx)
		require.Nil(t, err)
		require.Equal(t, []byte("message"), m.Payload)

		err = suite.Client.Ack(ctx, m)
		require.Nil(t, err)
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
				go func() {
					for msg := range sendc {
						err := suite.Client.Write(ctx, []byte(strconv.Itoa(msg)))
						require.Nil(t, err)
					}
				}()
			}

			for i := 0; i < test.readers; i++ {
				go func() {
					for {
						m, err := suite.Client.Read(ctx)
						if errors.Is(err, context.DeadlineExceeded) {
							return
						}

						require.Nil(t, err)
						receivec <- m.Payload

						err = suite.Client.Ack(ctx, m)
						require.Nil(t, err)
					}
				}()
			}

			for i := 0; i < test.messages; i++ {
				sendc <- i
			}
			close(sendc)

			received := make([]int, 0)
			for {
				select {
				case <-ctx.Done():
					require.Equal(t, test.messages, len(received))
					return
				case msg := <-receivec:
					i, err := strconv.Atoi(string(msg))
					require.Nil(t, err)

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
		err := alpha.Client.Write(ctx, []byte("message"))
		require.Nil(t, err)
	})

	t.Run("alpha read w/ no ack", func(t *testing.T) {
		m, err := alpha.Client.Read(ctx)
		require.Nil(t, err)
		require.Equal(t, []byte("message"), m.Payload)
	})

	t.Run("bravo read", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		_, err := bravo.Client.Read(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("bravo claim", func(t *testing.T) {
		m, err := bravo.Client.Claim(ctx, time.Second)
		require.Nil(t, err)
		require.Equal(t, []byte("message"), m.Payload)

		err = bravo.Client.Ack(ctx, m)
		require.Nil(t, err)
	})

	t.Run("alpha claim w/ empty stream", func(t *testing.T) {
		_, err := alpha.Client.Claim(ctx, time.Second)
		require.ErrorIs(t, err, ErrNoPending)
	})
}

func setup(ctx context.Context, t *testing.T) *suite {
	t.Helper()

	redis := redis.InitSuite(ctx, t)
	err := redis.Redis.FlushAll(ctx).Err()
	require.Nil(t, err)

	s := InitSuite(ctx, t)

	return &suite{Suite: *s}
}

type suite struct {
	Suite
}
