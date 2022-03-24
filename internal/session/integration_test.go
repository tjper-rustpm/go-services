// +build integration

package session

import (
	"context"
	"testing"
	"time"

	"github.com/tjper/rustcron/internal/redis"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.NewSession(ctx, t, "session-testing@gmail.com")

	t.Run("touch session that dne", func(t *testing.T) {
		_, err := suite.Manager.TouchSession(ctx, sess.ID)
		assert.ErrorIs(t, err, ErrSessionDNE, err.Error())
	})

	t.Run("delete session that dne", func(t *testing.T) {
		err := suite.Manager.DeleteSession(ctx, *sess)
		assert.Nil(t, err)
	})

	t.Run("retrieve session that dne", func(t *testing.T) {
		_, err := suite.Manager.RetrieveSession(ctx, sess.ID)
		assert.ErrorIs(t, err, ErrSessionDNE)
	})

	t.Run("create session", func(t *testing.T) {
		err := suite.Manager.CreateSession(ctx, *sess)
		assert.Nil(t, err)
	})

	t.Run("create session that already exists", func(t *testing.T) {
		err := suite.Manager.CreateSession(ctx, *sess)
		assert.ErrorIs(t, err, ErrSessionIDNotUnique)
	})

	t.Run("retrieve session", func(t *testing.T) {
		actual, err := suite.Manager.RetrieveSession(ctx, sess.ID)
		assert.Nil(t, err)
		assert.True(t, sess.Equal(*actual))
	})

	t.Run("touch session", func(t *testing.T) {
		sess, err := suite.Manager.TouchSession(ctx, sess.ID)
		assert.Nil(t, err)
		assert.WithinDuration(t, time.Now(), sess.LastActivityAt, time.Second)
	})

	t.Run("delete session", func(t *testing.T) {
		err := suite.Manager.DeleteSession(ctx, *sess)
		assert.Nil(t, err)
	})

	t.Run("create session", func(t *testing.T) {
		err := suite.Manager.CreateSession(ctx, *sess)
		assert.Nil(t, err)
	})

	t.Run("invalidate user's sessions", func(t *testing.T) {
		err := suite.Manager.InvalidateUserSessionsBefore(
			ctx,
			sess.User.ID,
			time.Now(),
		)
		assert.Nil(t, err)
	})

	t.Run("retrieve invalidated session", func(t *testing.T) {
		_, err := suite.Manager.RetrieveSession(ctx, sess.ID)
		assert.ErrorIs(t, err, ErrSessionDNE)
	})
}

func TestUpdateSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.NewSession(ctx, t, "session-testing@gmail.com")

	t.Run("create session", func(t *testing.T) {
		err := suite.Manager.CreateSession(ctx, *sess)
		assert.Nil(t, err)
	})

	t.Run("update session refreshed at", func(t *testing.T) {
		now := time.Now()
		updateFn := func(sess *Session) { sess.RefreshedAt = now }

		sess, err := suite.Manager.UpdateSession(ctx, sess.ID, updateFn)
		assert.Nil(t, err)
		assert.Equal(t, now, sess.RefreshedAt)
	})
}

func TestMarkStaleUserSessionsBefore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.NewSession(ctx, t, "session-testing@gmail.com")

	t.Run("create session", func(t *testing.T) {
		err := suite.Manager.CreateSession(ctx, *sess)
		assert.Nil(t, err)
	})

	t.Run("retrieve session", func(t *testing.T) {
		actual, err := suite.Manager.RetrieveSession(ctx, sess.ID)
		assert.Nil(t, err)
		assert.True(t, sess.Equal(*actual))
	})

	t.Run("mark session as stale", func(t *testing.T) {
		err := suite.Manager.MarkStaleUserSessionsBefore(ctx, sess.User.ID, time.Now())
		assert.Nil(t, err)
	})

	t.Run("retrieve session stale session", func(t *testing.T) {
		sess, err := suite.Manager.RetrieveSession(ctx, sess.ID)
		assert.ErrorIs(t, err, ErrSessionStale)
		assert.True(t, sess.Equal(*sess))
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
