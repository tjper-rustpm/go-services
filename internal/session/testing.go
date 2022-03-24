package session

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tjper/rustcron/internal/rand"
	"go.uber.org/zap"
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

	return &Suite{
		Manager: NewManager(zap.NewNop(), rdb, time.Hour),
	}
}

type Suite struct {
	Manager *Manager
}

func (s Suite) NewSession(ctx context.Context, t *testing.T, email string) *Session {
	t.Helper()

	id, err := rand.GenerateString(16)
	require.Nil(t, err)

	sess := New(
		id,
		User{
			ID:    uuid.New(),
			Email: email,
			Role:  RoleStandard,
		},
		time.Minute,
	)

	return sess
}

func (s Suite) CreateSession(ctx context.Context, t *testing.T, email string) *Session {
	t.Helper()

	sess := s.NewSession(ctx, t, email)

	err := s.Manager.CreateSession(ctx, *sess)
	require.Nil(t, err)

	return sess
}
