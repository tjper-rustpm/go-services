package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v72"
	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/internal/event"
	imodel "github.com/tjper/rustcron/internal/model"
	"github.com/tjper/rustcron/internal/stream"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestHandleVipRefreshEvent(t *testing.T) {
	dsn := os.Getenv("CRONMAN_DSN")
	migrations := os.Getenv("CRONMAN_MIGRATIONS")

	switch {
	case dsn == "":
		t.Skip("CRONMAN_DSN must be set to execute this test.")
	case migrations == "":
		t.Skip("CRONMAN_MIGRATIONS must be set to execute this test.")
	}

	type generated struct {
		serverID  uuid.UUID
		steamID   string
		expiresAt time.Time
	}
	type expected struct {
		err   error
		vips  model.Vips
		rcons []string
	}
	tests := map[string]struct {
		servers func(*generated) []interface{}
		events  func(*generated) []event.VipRefreshEvent
		exp     func(*generated) expected
	}{
		"SteamID empty": {
			servers: func(g *generated) []interface{} { return nil },
			events: func(g *generated) []event.VipRefreshEvent {
				g.expiresAt = time.Now().Add(24 * time.Hour).Round(time.Millisecond).UTC()

				return []event.VipRefreshEvent{
					event.NewVipRefreshEvent(g.serverID, "", g.expiresAt),
				}
			},
			exp: func(g *generated) expected {
				return expected{
					err: fmt.Errorf("refresh SteamID empty: %w", errNoRetry),
				}
			},
		},
		"ServerID empty": {
			servers: func(g *generated) []interface{} { return nil },
			events: func(g *generated) []event.VipRefreshEvent {
				g.steamID = uuid.NewString()
				g.expiresAt = time.Now().Add(24 * time.Hour).Round(time.Millisecond).UTC()

				return []event.VipRefreshEvent{
					event.NewVipRefreshEvent(uuid.Nil, g.steamID, g.expiresAt),
				}
			},
			exp: func(g *generated) expected {
				return expected{
					err: fmt.Errorf("refresh ServerID empty: %w", errNoRetry),
				}
			},
		},
		"ExpiresAt empty": {
			servers: func(g *generated) []interface{} { return nil },
			events: func(g *generated) []event.VipRefreshEvent {
				g.serverID = uuid.New()
				g.steamID = uuid.NewString()

				return []event.VipRefreshEvent{
					event.NewVipRefreshEvent(g.serverID, g.steamID, time.Time{}),
				}
			},
			exp: func(g *generated) expected {
				return expected{
					err: fmt.Errorf("refresh ExpiresAt empty: %w", errNoRetry),
				}
			},
		},
		"single dormant server, single event": {
			servers: func(g *generated) []interface{} {
				g.serverID = uuid.New()

				return []interface{}{
					&model.DormantServer{
						Server: model.Server{
							Model:        imodel.Model{ID: g.serverID},
							Name:         "server-name",
							InstanceID:   "instance-id",
							InstanceKind: model.InstanceKindStandard,
							AllocationID: "allocation-id",
							ElasticIP:    "elastic-ip",
							MaxPlayers:   200,
							MapSize:      3000,
							TickRate:     30,
							RconPassword: "rcon-password",
							Description:  "description",
							Background:   model.BackgroundKindAirport,
							URL:          "https://rustpm.com",
							BannerURL:    "https://rustpm.com/banner",
							Region:       model.RegionUsEast,
							Options:      map[string]interface{}{},
						},
					},
				}
			},
			events: func(g *generated) []event.VipRefreshEvent {
				g.steamID = uuid.NewString()
				g.expiresAt = time.Now().Add(24 * time.Hour).Round(time.Millisecond).UTC()

				return []event.VipRefreshEvent{
					event.NewVipRefreshEvent(g.serverID, g.steamID, g.expiresAt),
				}
			},
			exp: func(g *generated) expected {
				return expected{
					rcons: []string{},
					vips: model.Vips{
						{
							ServerID:  g.serverID,
							SteamID:   g.steamID,
							ExpiresAt: g.expiresAt,
						},
					},
				}
			},
		},
		"single dormant server, two events same steam IDs": {
			servers: func(g *generated) []interface{} {
				g.serverID = uuid.New()

				return []interface{}{
					&model.DormantServer{
						Server: model.Server{
							Model:        imodel.Model{ID: g.serverID},
							Name:         "server-name",
							InstanceID:   "instance-id",
							InstanceKind: model.InstanceKindStandard,
							AllocationID: "allocation-id",
							ElasticIP:    "elastic-ip",
							MaxPlayers:   200,
							MapSize:      3000,
							TickRate:     30,
							RconPassword: "rcon-password",
							Description:  "description",
							Background:   model.BackgroundKindAirport,
							URL:          "https://rustpm.com",
							BannerURL:    "https://rustpm.com/banner",
							Region:       model.RegionUsEast,
							Options:      map[string]interface{}{},
						},
					},
				}
			},
			events: func(g *generated) []event.VipRefreshEvent {
				g.steamID = uuid.NewString()
				g.expiresAt = time.Now().Add(24 * time.Hour).Round(time.Millisecond).UTC()

				return []event.VipRefreshEvent{
					event.NewVipRefreshEvent(g.serverID, g.steamID, time.Now().Round(time.Millisecond).UTC()),
					event.NewVipRefreshEvent(g.serverID, g.steamID, g.expiresAt),
				}
			},
			exp: func(g *generated) expected {
				return expected{
					rcons: []string{},
					vips: model.Vips{
						{
							ServerID:  g.serverID,
							SteamID:   g.steamID,
							ExpiresAt: g.expiresAt,
						},
					},
				}
			},
		},
		"one live server, single event": {
			servers: func(g *generated) []interface{} {
				g.serverID = uuid.New()

				return []interface{}{
					&model.LiveServer{
						AssociationID: "association-id",
						Server: model.Server{
							Model:        imodel.Model{ID: g.serverID},
							Name:         "server-name",
							InstanceID:   "instance-id",
							InstanceKind: model.InstanceKindStandard,
							AllocationID: "allocation-id",
							ElasticIP:    "elastic-ip",
							MaxPlayers:   200,
							MapSize:      3000,
							TickRate:     30,
							RconPassword: "rcon-password",
							Description:  "description",
							Background:   model.BackgroundKindAirport,
							URL:          "https://rustpm.com",
							BannerURL:    "https://rustpm.com/banner",
							Region:       model.RegionUsEast,
							Options:      map[string]interface{}{},
						},
					},
				}
			},
			events: func(g *generated) []event.VipRefreshEvent {
				g.steamID = uuid.NewString()
				g.expiresAt = time.Now().Add(24 * time.Hour).Round(time.Millisecond).UTC()

				return []event.VipRefreshEvent{
					event.NewVipRefreshEvent(g.serverID, g.steamID, g.expiresAt),
				}
			},
			exp: func(g *generated) expected {
				return expected{
					rcons: []string{
						fmt.Sprintf("elastic-ip:28016 rcon-password %s %s", g.steamID, rcon.VipGroup),
					},
					vips: model.Vips{
						{
							ServerID:  g.serverID,
							SteamID:   g.steamID,
							ExpiresAt: g.expiresAt,
						},
					},
				}
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			g := new(generated)

			// Establish DB connection and perform necessary migrations before
			// testing VipRefreshEvent handling.
			store := newDB(t, dsn, migrations)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			for _, server := range test.servers(g) {
				server := server
				createServer(ctx, t, store, server)
				defer func() {
					store.WithContext(ctx).Unscoped().Delete(server)
				}()
			}

			rcon := rcon.NewHubMock()

			handler := NewHandler(
				zap.NewNop(),
				store,
				stream.NewClientMock(),
				rcon,
			)
			for _, event := range test.events(g) {
				err := handler.handleVipRefreshEvent(ctx, &event)

				expected := test.exp(g).err
				if expected != nil {
					require.EqualError(t, expected, err.Error())
				}
			}

			for _, expected := range test.exp(g).rcons {
				actual := rcon.LPop()
				require.Equal(t, expected, actual)
			}

			vips, err := db.ListVipsByServerID(ctx, store, g.serverID)
			require.Nil(t, err)

			for i := range vips {
				vips[i].Model.Scrub()
			}
			expected := test.exp(g).vips

			require.True(t, expected.Equal(vips), "expected: %v\nactual: %v", expected, vips)
		})
	}
}

func TestLaunch(t *testing.T) {
	// NOTE: The database related environment variables below are needed to
	// initialize the stream.Handler, but the database is not actually used for
	// this test. This is one downside to not isolating database interactions
	// within single package and type.
	dsn := os.Getenv("CRONMAN_DSN")
	migrations := os.Getenv("CRONMAN_MIGRATIONS")

	switch {
	case dsn == "":
		t.Skip("CRONMAN_DSN must be set to execute this test.")
	case migrations == "":
		t.Skip("CRONMAN_MIGRATIONS must be set to execute this test.")
	}

	readc := make(chan stream.Message)
	ackc := make(chan struct{})

	streamClient := stream.NewClientMock(
		stream.WithClaim(func(context.Context, time.Duration) (*stream.Message, error) {
			return nil, stream.ErrNoPending
		}),
		stream.WithRead(func(context.Context) (*stream.Message, error) {
			m := <-readc
			return &m, nil
		}),
		stream.WithAck(func(_ context.Context, _ *stream.Message) error {
			ackc <- struct{}{}
			return nil
		}),
	)
	handler := NewHandler(
		zap.NewNop(),
		newDB(t, dsn, migrations),
		streamClient,
		rcon.NewHubMock(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		err := handler.Launch(ctx)
		require.ErrorIs(t, err, context.Canceled)
	}()

	// Build a stream.Message for Launch to handle.
	event := event.NewStripeWebhookEvent(stripe.Event{})
	b, err := json.Marshal(&event)
	require.Nil(t, err)

	readc <- stream.Message{
		ID:      uuid.NewString(),
		Payload: b,
	}

	select {
	case <-ctx.Done():
		require.FailNow(t, "Context should not be done before an ack occurs.")
	case <-ackc:
		break
	}
}

// newDB is a helper for creating a new db connection instance within a test.
func newDB(t *testing.T, dsn, migrations string) *gorm.DB {
	t.Helper()

	dbconn, err := db.Open(dsn)
	require.Nil(t, err)

	err = db.Migrate(dbconn, migrations)
	require.Nil(t, err)

	return dbconn
}

// createServer creates the passed server within the passed db.
func createServer(ctx context.Context, t *testing.T, db *gorm.DB, server interface{}) {
	t.Helper()

	err := db.WithContext(ctx).Create(server).Error
	require.Nil(t, err)
}
