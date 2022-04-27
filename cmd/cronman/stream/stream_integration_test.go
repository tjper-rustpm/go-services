//go:build integration
// +build integration

package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/internal/event"
	"github.com/tjper/rustcron/internal/gorm"
	"github.com/tjper/rustcron/internal/stream"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestInvoicePaidEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize test data
	initializer := newInitializer(t)
	initializer.run(ctx, t)

	type data struct {
		subscription uuid.UUID
		server       uuid.UUID
		steam        string
		expiresAt    time.Time
	}
	alpha := data{
		subscription: uuid.New(),
		server:       initializer.alpha.Server.ID,
		steam:        "alpha-steam-id",
		expiresAt:    time.Now().Add(time.Hour * 24 * 31), // 31 days in the future
	}
	bravo := data{
		subscription: uuid.New(),
		server:       initializer.alpha.Server.ID,
		steam:        "bravo-steam-id",
		expiresAt:    time.Now().Add(time.Hour * 24 * 31), // 31 days in the future
	}
	charlie := data{
		subscription: uuid.New(),
		server:       initializer.charlie.Server.ID,
		steam:        "charlie-steam-id",
		expiresAt:    time.Now().Add(time.Hour * 24 * 31), // 31 days in the future
	}
	live := data{
		subscription: uuid.New(),
		server:       initializer.live.Server.ID,
		steam:        "live-steam-id",
		expiresAt:    time.Now().Add(time.Hour * 24 * 31), // 31 days in the future
	}

	type expected struct {
		vips  model.Vips
		rcons []string
	}
	stages := []struct {
		serverID uuid.UUID
		events   []event.InvoicePaidEvent
		exp      expected
	}{
		{
			serverID: alpha.server,
			events: []event.InvoicePaidEvent{
				event.NewInvoicePaidEvent(alpha.subscription, alpha.server, alpha.steam),
			},
			exp: expected{
				vips: model.Vips{
					{
						SubscriptionID: alpha.subscription,
						ServerID:       alpha.server,
						SteamID:        alpha.steam,
						ExpiresAt:      alpha.expiresAt,
					},
				},
			},
		},
		{
			serverID: bravo.server,
			events: []event.InvoicePaidEvent{
				event.NewInvoicePaidEvent(bravo.subscription, bravo.server, bravo.steam),
			},
			exp: expected{
				vips: model.Vips{
					{
						SubscriptionID: alpha.subscription,
						ServerID:       alpha.server,
						SteamID:        alpha.steam,
						ExpiresAt:      alpha.expiresAt,
					},
					{
						SubscriptionID: bravo.subscription,
						ServerID:       bravo.server,
						SteamID:        bravo.steam,
						ExpiresAt:      bravo.expiresAt,
					},
				},
			},
		},
		{
			serverID: initializer.alpha.Server.ID,
			events: []event.InvoicePaidEvent{
				event.NewInvoicePaidEvent(charlie.subscription, charlie.server, charlie.steam),
			},
			exp: expected{
				vips: model.Vips{
					{
						SubscriptionID: alpha.subscription,
						ServerID:       alpha.server,
						SteamID:        alpha.steam,
						ExpiresAt:      alpha.expiresAt,
					},
					{
						SubscriptionID: bravo.subscription,
						ServerID:       bravo.server,
						SteamID:        bravo.steam,
						ExpiresAt:      bravo.expiresAt,
					},
				},
			},
		},
		{
			serverID: initializer.live.Server.ID,
			events: []event.InvoicePaidEvent{
				event.NewInvoicePaidEvent(live.subscription, live.server, live.steam),
			},
			exp: expected{
				vips: model.Vips{
					{
						SubscriptionID: live.subscription,
						ServerID:       live.server,
						SteamID:        live.steam,
						ExpiresAt:      live.expiresAt,
					},
				},
				rcons: []string{
					fmt.Sprintf(
						"%s:28016 %s %s %s",
						initializer.live.Server.ElasticIP,
						initializer.live.Server.RconPassword,
						live.steam,
						rcon.BypassQueueAllow,
					),
				},
			},
		},
	}

	// Initialize test dependencies.
	deps := newDeps(ctx, t)

	// Launch stream handler in own goroutine.
	go func() {
		err := deps.handler.Launch(ctx)
		require.ErrorIs(t, err, context.Canceled)
	}()

	for i, test := range stages {
		t.Run(fmt.Sprintf("stage - %d", i), func(t *testing.T) {
			// Marshal and write events to stream.
			for _, event := range test.events {
				b, err := json.Marshal(event)
				require.Nil(t, err)

				err = deps.stream.Write(ctx, b)
				require.Nil(t, err)
			}

      // Sleep is lazy, but simple and signaling when the written message has
      // been processed seems like more work than it's worth.
			time.Sleep(200 * time.Millisecond)

			// Check if store is in expected state.
			var vips model.Vips
			err := deps.store.FindByServerID(ctx, &vips, test.serverID)
			require.Nil(t, err)

			require.Len(t, vips, len(test.exp.vips), "actual vips different length than expected vips")

			// Check all non deterministic fields, and set them to equal to allow for
			// object level comparisons later in the function.
			for i := range test.exp.vips {
				actual := vips[i]
				require.NotEmpty(t, actual.ID)
				require.WithinDuration(t, time.Now(), actual.UpdatedAt, time.Second)
				require.WithinDuration(t, time.Now(), actual.CreatedAt, time.Second)
				require.WithinDuration(t, test.exp.vips[i].ExpiresAt, actual.ExpiresAt, time.Second)

				test.exp.vips[i].ID = actual.ID
				test.exp.vips[i].ExpiresAt = actual.ExpiresAt
				test.exp.vips[i].UpdatedAt = actual.UpdatedAt
				test.exp.vips[i].CreatedAt = actual.CreatedAt
			}

			require.Equal(t, test.exp.vips, vips)

			for _, expected := range test.exp.rcons {
				actual := deps.rconHub.LPop()
				require.Equal(t, expected, actual)
			}
		})
	}
}

func newDeps(ctx context.Context, t *testing.T) *deps {
	t.Helper()

	// Configure logger for testing suite.
	logger := zap.NewNop()

	// Configure redis for testing suite.
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
	err = rdb.FlushAll(ctx).Err()
	require.Nil(t, err)

	// Configure stream Client for testing suite.
	stream, err := stream.Init(ctx, logger, rdb, "cronman-stream-integration-test")
	require.Nil(t, err)

	// Configure database store for testing suite.
	const (
		dsn        = "host=db user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=UTC"
		migrations = "file://../db/migrations"
	)

	dbconn, err := db.Open(dsn)
	require.Nil(t, err)

	err = db.Migrate(dbconn, migrations)
	require.Nil(t, err)

	store := gorm.NewStore(dbconn)

	rconHub := rcon.NewHubMock()

	// Configure stream Handler for testing suite.
	handler := NewHandler(
		logger,
		store,
		stream,
		rconHub,
	)

	return &deps{
		handler: handler,
		stream:  stream,
		store:   store,
		rconHub: rconHub,
	}
}

type deps struct {
	handler *Handler
	stream  *stream.Client
	store   *gorm.Store
	rconHub *rcon.HubMock
}

func newInitializer(t *testing.T) *initializer {
	t.Helper()

	// Configure database store.
	const (
		dsn        = "host=db user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=UTC"
		migrations = "file://../db/migrations"
	)

	dbconn, err := db.Open(dsn)
	require.Nil(t, err)

	err = db.Migrate(dbconn, migrations)
	require.Nil(t, err)

	store := gorm.NewStore(dbconn)

	return &initializer{
		store: store,
		once:  new(sync.Once),
	}
}

type initializer struct {
	store *gorm.Store

	once    *sync.Once
	alpha   *model.DormantServer
	charlie *model.DormantServer
	live    *model.LiveServer
}

func (i *initializer) run(ctx context.Context, t *testing.T) {
	t.Helper()

	i.once.Do(func() {
		dormants := []**model.DormantServer{
			&i.alpha,
			&i.charlie,
		}

		for _, dormant := range dormants {
			server := &model.DormantServer{
				Server: model.Server{
					Name:         "DormantServer",
					InstanceKind: model.InstanceKindStandard,
					MaxPlayers:   200,
					MapSize:      4000,
					TickRate:     30,
					RconPassword: "rcon-password",
					Description:  "Server for cronman stream integration testing.",
					URL:          "https://rustpm.com",
					Background:   model.BackgroundKindOxum,
					BannerURL:    "https://rustpm.com/banner",
					Region:       model.RegionUsEast,
					Wipes:        model.Wipes{model.Wipe{MapSeed: 2000, MapSalt: 2000}},
				},
			}

			err := i.store.Create(ctx, server)
			require.Nil(t, err)

			*dormant = server
		}

		live := &model.LiveServer{
			AssociationID: "associated-id",
			ActivePlayers: 0,
			QueuedPlayers: 0,
			Server: model.Server{
				Name:         "DormantServer",
				InstanceKind: model.InstanceKindStandard,
				MaxPlayers:   200,
				MapSize:      4000,
				TickRate:     30,
				ElasticIP:    "192.168.0.1",
				RconPassword: "rcon-password",
				Description:  "Server for cronman stream integration testing.",
				URL:          "https://rustpm.com",
				Background:   model.BackgroundKindOxum,
				BannerURL:    "https://rustpm.com/banner",
				Region:       model.RegionUsEast,
				Wipes:        model.Wipes{model.Wipe{MapSeed: 2000, MapSalt: 2000}},
			},
		}

		err := i.store.Create(ctx, live)
		require.Nil(t, err)

		i.live = live
	})
}
