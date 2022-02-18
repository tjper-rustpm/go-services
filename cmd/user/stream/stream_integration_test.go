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
	"github.com/tjper/rustcron/internal/session"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestHandleSubscriptionCreated(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := setup(ctx, t)

	var user model.User
	t.Run("create user", func(t *testing.T) {
		user = s.createUser(t, "subscription-create-user@gmail.com")
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
		assert.Nil(t, res.Error)

		assert.Len(t, actual.Subscriptions, 1)
		assert.Equal(t, subscriptionID, actual.Subscriptions[0].SubscriptionID)
		assert.Equal(t, serverID, actual.Subscriptions[0].ServerID)

		// session has been marked stale
		staleAt := s.Sessions.StaleAt(user.ID)
		assert.WithinDuration(t, time.Now(), staleAt, time.Second)
	})
}

func TestHandleSubscriptionDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := setup(ctx, t)

	var user model.User
	subscriptionID := uuid.New()
	t.Run("create user", func(t *testing.T) {
		user = s.createUser(t, "subscription-delete-user@gmail.com")
		user.Subscriptions = []model.Subscription{
			{SubscriptionID: subscriptionID, ServerID: uuid.New()},
		}

		res := s.db.Save(&user)
		assert.Nil(t, res.Error)
	})

	t.Run("handle subscription delete event", func(t *testing.T) {
		e := event.NewSubscriptionDeleteEvent(subscriptionID)

		s.writeEvent(ctx, t, e)

		var actual model.User
		res := s.db.Preload(clause.Associations).First(&actual, user.ID)
		assert.Nil(t, res.Error)

		assert.Empty(t, actual.Subscriptions)

		staleAt := s.Sessions.StaleAt(user.ID)
		assert.WithinDuration(t, time.Now(), staleAt, time.Second)
	})
}

func setup(ctx context.Context, t *testing.T) *suite {
	t.Helper()

	s := integration.InitSuite(ctx, t, integration.WithLogger(zap.NewExample()))

	const (
		dsn        = "host=db user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=UTC"
		migrations = "file://../db/migrations"
	)

	dbconn, err := db.Open(dsn)
	require.Nil(t, err)

	err = db.Migrate(dbconn, migrations)
	require.Nil(t, err)

	handler := NewHandler(s.Logger, s.Stream, dbconn, s.Sessions)

	go func() {
		err := handler.Launch(ctx)
		assert.ErrorIs(t, err, context.Canceled)
	}()

	return &suite{
		Suite:   *s,
		db:      dbconn,
		handler: handler,
	}
}

type suite struct {
	integration.Suite
	db      *gorm.DB
	handler *Handler
}

func (s suite) createUser(t *testing.T, email string) model.User {
	t.Helper()

	user := model.User{
		Email:              email,
		Password:           []byte("test-password"),
		Salt:               "test-salt",
		Role:               session.RoleStandard,
		VerificationHash:   uuid.New().String(),
		VerificationSentAt: time.Now(),
	}

	res := s.db.Create(&user)
	assert.Nil(t, res.Error)

	return user
}

func (s suite) writeEvent(ctx context.Context, t *testing.T, e interface{}) {
	t.Helper()

	b, err := json.Marshal(&e)
	assert.Nil(t, err)

	err = s.Stream.Write(ctx, b)
	assert.Nil(t, err)

	time.Sleep(100 * time.Millisecond) // wait to allow handler to process event
}
