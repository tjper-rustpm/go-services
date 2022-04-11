//go:build integration
// +build integration

package stream

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/user/db"
	"github.com/tjper/rustcron/cmd/user/model"
	"github.com/tjper/rustcron/internal/event"
	"github.com/tjper/rustcron/internal/integration"
	"github.com/tjper/rustcron/internal/rand"
	"github.com/tjper/rustcron/internal/redis"
	"github.com/tjper/rustcron/internal/session"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestHandleSubscriptionCreated(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := setup(ctx, t)

	var (
		user model.User
		sess session.Session
	)
	t.Run("setup", func(t *testing.T) {
		user = s.createUser(t, "subscription-create-user@gmail.com")

		sess = *s.sessions.NewSession(ctx, t, "subscription-create-user@gmail.com", session.RoleStandard)
		sess.User.ID = user.ID

		err := s.sessions.Manager.CreateSession(ctx, sess)
		require.Nil(t, err)
	})

	t.Run("handle subscription create event", func(t *testing.T) {
		subscriptionID := uuid.New()
		serverID := uuid.New()
		e := event.NewSubscriptionCreatedEvent(
			subscriptionID,
			user.ID,
			serverID,
		)

		s.writeEvent(ctx, t, e)

		// susbcription is on user
		var actual model.User
		res := s.db.Preload(clause.Associations).First(&actual, user.ID)
		require.Nil(t, res.Error)

		require.Len(t, actual.Subscriptions, 1)
		require.Equal(t, subscriptionID, actual.Subscriptions[0].SubscriptionID)
		require.Equal(t, serverID, actual.Subscriptions[0].ServerID)

		// session has been marked stale
		_, err := s.sessions.Manager.RetrieveSession(ctx, sess.ID)
		require.ErrorIs(t, err, session.ErrSessionStale)
	})
}

func TestHandleSubscriptionDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := setup(ctx, t)

	var (
		subscriptionID = uuid.New()
		user           model.User
		sess           session.Session
	)
	t.Run("setup", func(t *testing.T) {
		user = s.createUser(t, "subscription-delete-user@gmail.com")
		user.Subscriptions = []model.Subscription{
			{SubscriptionID: subscriptionID, ServerID: uuid.New()},
		}

		res := s.db.Save(&user)
		require.Nil(t, res.Error)

		sess = *s.sessions.NewSession(ctx, t, "subscription-delete-user@gmail.com", session.RoleStandard)
		sess.User.ID = user.ID

		err := s.sessions.Manager.CreateSession(ctx, sess)
		require.Nil(t, err)
	})

	t.Run("handle subscription delete event", func(t *testing.T) {
		e := event.NewSubscriptionDeleteEvent(subscriptionID)

		s.writeEvent(ctx, t, e)

		var actual model.User
		res := s.db.Preload(clause.Associations).First(&actual, user.ID)
		require.Nil(t, res.Error)

		require.Empty(t, actual.Subscriptions)

		_, err := s.sessions.Manager.RetrieveSession(ctx, sess.ID)
		require.ErrorIs(t, err, session.ErrSessionStale)
	})
}

func setup(ctx context.Context, t *testing.T) *suite {
	t.Helper()

	redis := redis.InitSuite(ctx, t)
	err := redis.Redis.FlushAll(ctx).Err()
	require.Nil(t, err)

	s := integration.InitSuite(ctx, t, integration.WithLogger(zap.NewExample()))
	sessions := session.InitSuite(ctx, t)

	const (
		dsn        = "host=db user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=UTC"
		migrations = "file://../db/migrations"
	)

	dbconn, err := db.Open(dsn)
	require.Nil(t, err)

	err = db.Migrate(dbconn, migrations)
	require.Nil(t, err)

	handler := NewHandler(s.Logger, s.Stream, dbconn, sessions.Manager)

	go func() {
		err := handler.Launch(ctx)
		require.ErrorIs(t, err, context.Canceled)
	}()

	return &suite{
		Suite:    *s,
		sessions: sessions,
		db:       dbconn,
		handler:  handler,
	}
}

type suite struct {
	integration.Suite
	sessions *session.Suite

	db      *gorm.DB
	handler *Handler
}

func (s suite) createUser(t *testing.T, email string) model.User {
	t.Helper()

	salt, err := rand.GenerateString(32)
	require.Nil(t, err)

	hash, err := rand.GenerateString(32)
	require.Nil(t, err)

	user := model.User{
		Email:              email,
		Password:           []byte("test-password"),
		Salt:               salt,
		Role:               session.RoleStandard,
		VerificationHash:   hash,
		VerificationSentAt: time.Now(),
	}

	res := s.db.Create(&user)
	require.Nil(t, res.Error)

	return user
}

func (s suite) writeEvent(ctx context.Context, t *testing.T, e interface{}) {
	t.Helper()

	b, err := json.Marshal(&e)
	require.Nil(t, err)

	err = s.Stream.Write(ctx, b)
	require.Nil(t, err)

	time.Sleep(100 * time.Millisecond) // wait to allow handler to process event
}
