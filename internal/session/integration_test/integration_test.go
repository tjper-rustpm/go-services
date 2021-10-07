// +build integration

package session

import (
	"context"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/config"
	"github.com/tjper/rustcron/internal/session"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := &suite{}
	s.setup(ctx, t)

	t.Run("save session that does not exist", func(t *testing.T) {
		s.save(ctx, t, session.Session{ID: "session-id"}, session.ErrSessionDNE)
	})
	t.Run("delete session that does not exist", func(t *testing.T) {
		s.delete(ctx, t, "session-id", nil)
	})
	t.Run("retrieve session that does not exist", func(t *testing.T) {
		s.retrieve(ctx, t, "session-id", nil, session.ErrSessionDNE)
	})
	t.Run("create session", func(t *testing.T) {
		s.create(ctx, t, session.Session{ID: "session-id"}, nil)
	})
	t.Run("create session that already exists", func(t *testing.T) {
		s.create(ctx, t, session.Session{ID: "session-id"}, session.ErrSessionIDNotUnique)
	})
	t.Run("retrieve session", func(t *testing.T) {
		s.retrieve(ctx, t, "session-id", &session.Session{ID: "session-id"}, nil)
	})
	t.Run("save session", func(t *testing.T) {
		s.save(ctx, t, session.Session{ID: "session-id"}, nil)
	})
	t.Run("delete session", func(t *testing.T) {
		s.delete(ctx, t, "session-id", nil)
	})
}

// --- suite ---

type suite struct {
	manager *session.Manager
}

func (s *suite) setup(ctx context.Context, t *testing.T) {
	cfg := config.Load()
	redis := redis.NewClient(
		&redis.Options{
			Addr:     cfg.RedisAddr(),
			Password: cfg.RedisPassword(),
		},
	)
	err := redis.Ping(ctx).Err()
	require.Nil(t, err)

	s.manager = session.NewManager(redis)
}

func (s *suite) create(
	ctx context.Context,
	t *testing.T,
	sess session.Session,
	expErr error,
) {
	err := s.manager.Create(ctx, sess)
	assert.Equal(t, expErr, err)
}

func (s *suite) retrieve(
	ctx context.Context,
	t *testing.T,
	sessionID string,
	expSession *session.Session,
	expErr error,
) {
	sess, err := s.manager.Retrieve(ctx, sessionID)
	assert.Equal(t, expErr, err)
	if err != nil {
		return
	}
	assert.True(t, expSession.Equal(*sess), "sessions are not equal")
}

func (s *suite) save(
	ctx context.Context,
	t *testing.T,
	sess session.Session,
	expErr error,
) {
	err := s.manager.Save(ctx, sess)
	assert.Equal(t, expErr, err)
}

func (s *suite) delete(
	ctx context.Context,
	t *testing.T,
	sessionID string,
	expErr error,
) {
	err := s.manager.Delete(ctx, sessionID)
	assert.Equal(t, expErr, err)
}
