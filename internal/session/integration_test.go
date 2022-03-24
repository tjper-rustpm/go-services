//go:build integration
// +build integration

package session

import (
	"context"
	"testing"
	"time"

	"github.com/tjper/rustcron/internal/redis"

	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.NewSession(ctx, t, "session-testing@gmail.com")

	t.Run("touch session that dne", func(t *testing.T) {
		_, err := suite.Manager.TouchSession(ctx, sess.ID)
		require.ErrorIs(t, err, ErrSessionDNE, err.Error())
	})

	t.Run("delete session that dne", func(t *testing.T) {
		err := suite.Manager.DeleteSession(ctx, *sess)
		require.Nil(t, err)
	})

	t.Run("retrieve session that dne", func(t *testing.T) {
		_, err := suite.Manager.RetrieveSession(ctx, sess.ID)
		require.ErrorIs(t, err, ErrSessionDNE)
	})

	t.Run("create session", func(t *testing.T) {
		err := suite.Manager.CreateSession(ctx, *sess)
		require.Nil(t, err)
	})

	t.Run("create session that already exists", func(t *testing.T) {
		err := suite.Manager.CreateSession(ctx, *sess)
		require.ErrorIs(t, err, ErrSessionIDNotUnique)
	})

	t.Run("retrieve session", func(t *testing.T) {
		actual, err := suite.Manager.RetrieveSession(ctx, sess.ID)
		require.Nil(t, err)
		require.True(t, sess.Equal(*actual))
	})

	t.Run("touch session", func(t *testing.T) {
		sess, err := suite.Manager.TouchSession(ctx, sess.ID)
		require.Nil(t, err)
		require.WithinDuration(t, time.Now(), sess.LastActivityAt, time.Second)
	})

	t.Run("delete session", func(t *testing.T) {
		err := suite.Manager.DeleteSession(ctx, *sess)
		require.Nil(t, err)
	})

	t.Run("create session", func(t *testing.T) {
		err := suite.Manager.CreateSession(ctx, *sess)
		require.Nil(t, err)
	})

	t.Run("invalidate user's sessions", func(t *testing.T) {
		err := suite.Manager.InvalidateUserSessionsBefore(
			ctx,
			sess.User.ID,
			time.Now(),
		)
		require.Nil(t, err)
	})

	t.Run("retrieve invalidated session", func(t *testing.T) {
		_, err := suite.Manager.RetrieveSession(ctx, sess.ID)
		require.ErrorIs(t, err, ErrSessionDNE)
	})
}

func TestUpdateSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.NewSession(ctx, t, "session-testing@gmail.com")

	t.Run("create session", func(t *testing.T) {
		err := suite.Manager.CreateSession(ctx, *sess)
		require.Nil(t, err)
	})

	t.Run("update session refreshed at", func(t *testing.T) {
		now := time.Now()
		updateFn := func(sess *Session) { sess.RefreshedAt = now }

		sess, err := suite.Manager.UpdateSession(ctx, sess.ID, updateFn)
		require.Nil(t, err)
		require.Equal(t, now, sess.RefreshedAt)
	})
}

func TestMarkStaleUserSessionsBefore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.NewSession(ctx, t, "session-testing@gmail.com")

	t.Run("create session", func(t *testing.T) {
		err := suite.Manager.CreateSession(ctx, *sess)
		require.Nil(t, err)
	})

	t.Run("retrieve session", func(t *testing.T) {
		actual, err := suite.Manager.RetrieveSession(ctx, sess.ID)
		require.Nil(t, err)
		require.True(t, sess.Equal(*actual))
	})

	t.Run("mark session as stale", func(t *testing.T) {
		err := suite.Manager.MarkStaleUserSessionsBefore(ctx, sess.User.ID, time.Now())
		require.Nil(t, err)
	})

	t.Run("retrieve session stale session", func(t *testing.T) {
		sess, err := suite.Manager.RetrieveSession(ctx, sess.ID)
		require.ErrorIs(t, err, ErrSessionStale)
		require.True(t, sess.Equal(*sess))
	})
}

// --- suite ---

type suite struct {
	Suite
}

func setup(ctx context.Context, t *testing.T) *suite {
	t.Helper()

	redis := redis.InitSuite(ctx, t)
	err := redis.Redis.FlushAll(ctx).Err()
	require.Nil(t, err)

	s := InitSuite(ctx, t)

	return &suite{
		Suite: *s,
	}
}
