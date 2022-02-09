// +build sessionintegration

package session

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/tjper/rustcron/internal/rand"
	"github.com/tjper/rustcron/internal/session"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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

	t.Run("touch session that dne", func(t *testing.T) {
		_, err := suite.manager.TouchSession(ctx, suite.session.ID, time.Hour)
		assert.ErrorIs(t, err, session.ErrSessionDNE, err.Error())
	})

	t.Run("delete session that dne", func(t *testing.T) {
		err := suite.manager.DeleteSession(ctx, suite.session)
		assert.Nil(t, err)
	})

	t.Run("retrieve session that dne", func(t *testing.T) {
		_, err := suite.manager.RetrieveSession(ctx, suite.session.ID)
		assert.ErrorIs(t, err, session.ErrSessionDNE)
	})

	t.Run("create session", func(t *testing.T) {
		err := suite.manager.CreateSession(ctx, suite.session, time.Hour)
		assert.Nil(t, err)
	})

	t.Run("create session that already exists", func(t *testing.T) {
		err := suite.manager.CreateSession(ctx, suite.session, time.Hour)
		assert.ErrorIs(t, err, session.ErrSessionIDNotUnique)
	})

	t.Run("retrieve session", func(t *testing.T) {
		sess, err := suite.manager.RetrieveSession(ctx, suite.session.ID)
		assert.Nil(t, err)
		assert.True(t, suite.session.Equal(*sess))
	})

	time.Sleep(time.Second)
	t.Run("touch session", func(t *testing.T) {
		sess, err := suite.manager.TouchSession(ctx, suite.session.ID, time.Hour)
		assert.Nil(t, err)
		assert.WithinDuration(t, time.Now(), sess.LastActivityAt, time.Second)
	})

	t.Run("delete session", func(t *testing.T) {
		err := suite.manager.DeleteSession(ctx, suite.session)
		assert.Nil(t, err)
	})

	t.Run("create session", func(t *testing.T) {
		err := suite.manager.CreateSession(ctx, suite.session, time.Hour)
		assert.Nil(t, err)
	})

	t.Run("invalidate user's sessions", func(t *testing.T) {
		err := suite.manager.InvalidateUserSessionsBefore(
			ctx,
			suite.session.User.ID,
			time.Now(),
		)
		assert.Nil(t, err)
	})

	t.Run("retrieve invalidated session", func(t *testing.T) {
		_, err := suite.manager.RetrieveSession(ctx, suite.session.ID)
		assert.ErrorIs(t, err, session.ErrSessionDNE)
	})

}

func TestAddRemoveSessionVIPs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	t.Run("create session", func(t *testing.T) {
		err := suite.manager.CreateSession(ctx, suite.session, time.Hour)
		assert.Nil(t, err)
	})

	vips := []uuid.UUID{uuid.New(), uuid.New()}

	t.Run("add session VIPs", func(t *testing.T) {
		updateFn := func(sess *session.Session) { sess.User.VIPs = vips }

		sess, err := suite.manager.UpdateSession(ctx, suite.session.ID, updateFn, time.Hour)
		assert.Nil(t, err)
		assert.Equal(t, vips, sess.User.VIPs)
	})

	t.Run("retrieve session w/ VIPs", func(t *testing.T) {
		sess, err := suite.manager.RetrieveSession(ctx, suite.session.ID)
		assert.Nil(t, err)
		assert.Equal(t, vips, sess.User.VIPs)
	})

	t.Run("remove session VIPs", func(t *testing.T) {
		updateFn := func(sess *session.Session) { sess.User.VIPs = nil }

		sess, err := suite.manager.UpdateSession(ctx, suite.session.ID, updateFn, time.Hour)
		assert.Nil(t, err)
		assert.Nil(t, sess.User.VIPs)
	})

	t.Run("retrieve session w/o VIPs", func(t *testing.T) {
		sess, err := suite.manager.RetrieveSession(ctx, suite.session.ID)
		assert.Nil(t, err)
		assert.Nil(t, sess.User.VIPs)
	})
}

// --- suite ---

type suite struct {
	manager *session.Manager

	session session.Session
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

	id, err := rand.GenerateString(16)
	require.Nil(t, err)

	return &suite{
		manager: session.NewManager(zap.NewNop(), redis),
		session: session.Session{
			ID: id,
			User: session.User{
				ID:    uuid.New(),
				Email: "fake@email.com",
				Role:  session.RoleStandard,
			},
			AbsoluteExpiration: time.Now().UTC().Add(time.Hour),
			LastActivityAt:     time.Now().UTC(),
			CreatedAt:          time.Now().UTC(),
		},
	}
}
