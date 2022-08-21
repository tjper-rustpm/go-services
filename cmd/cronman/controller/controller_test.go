package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/internal/gorm"
	imodel "github.com/tjper/rustcron/internal/model"

	"go.uber.org/zap"
)

func TestLiveServerRconForEach(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		count int
	}{
		{name: "single live server", count: 1},
		{name: "two live servers", count: 2},
		{name: "five live servers", count: 5},
		{name: "one-hundred live servers", count: 100},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			servers := newLiveServers(test.count)
			expected := make(model.LiveServers, len(servers))
			copy(expected, servers)

			finder := newFinderMock(servers)

			controller := &Controller{
				logger: zap.NewNop(),
				finder: finder,
				hub:    rcon.NewHubMock(),
			}

			fn := func(_ context.Context, server model.LiveServer, _ rcon.IRcon) error {
				expect := expected[0]
				require.Equal(t, expect.Model.ID, server.Model.ID)
				expected = expected[1:]
				return nil
			}

			err := controller.LiveServerRconForEach(ctx, fn)
			require.Nil(t, err)
		})
	}
}

func TestCaptureServerInfo(t *testing.T) {
	t.Parallel()

	type expected struct {
		changes interface{}
	}
	tests := []struct {
		name       string
		serverInfo rcon.ServerInfo
		server     model.LiveServer
		exp        expected
	}{
		{
			name: "happy path",
			serverInfo: rcon.ServerInfo{
				Players: 101,
				Queued:  5,
			},
			server: model.LiveServer{},
			exp: expected{
				changes: map[string]interface{}{
					"active_players": 101,
					"queued_players": 5,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			updater := newUpdaterMock(test.exp.changes)

			controller := &Controller{
				logger:  zap.NewNop(),
				updater: updater,
			}

			hub := rcon.NewHubMock(rcon.WithServerInfo(test.serverInfo))
			rcon, err := hub.Dial(ctx, "test-ip", "test-password")
			require.Nil(t, err)

			err = controller.CaptureServerInfo(ctx, test.server, rcon)
			require.Nil(t, err)
		})
	}
}

func TestSayServerTimeRemaining(t *testing.T) {
	t.Parallel()

	type expected struct {
		said string
	}
	tests := []struct {
		name       string
		serverName string
		duration   time.Duration
		exp        expected
	}{
		{
			name:       "1 minute",
			serverName: "Rustpm Test Server",
			duration:   2 * time.Minute,
			exp: expected{
				said: "Rustpm Test Server will be offline in 1 minute. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
		{
			name:       "59 minutes",
			serverName: "Rustpm Test Server",
			duration:   time.Hour,
			exp: expected{
				said: "Rustpm Test Server will be offline in 59 minutes. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
		{
			name:       "1 hour",
			serverName: "Rustpm Test Server",
			duration:   time.Hour + time.Minute,
			exp: expected{
				said: "Rustpm Test Server will be offline in 1 hour. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
		{
			name:       "1 hour and 30 minutes",
			serverName: "Rustpm Test Server",
			duration:   time.Hour + 30*time.Minute,
			exp: expected{
				said: "Rustpm Test Server will be offline in 1 hour and 30 minutes. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
		{
			name:       "2 hours",
			serverName: "Rustpm Test Server",
			duration:   2*time.Hour + time.Minute,
			exp: expected{
				said: "Rustpm Test Server will be offline in 2 hours. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
		{
			name:       "2 hours and 1 minute",
			serverName: "Rustpm Test Server",
			duration:   2*time.Hour + 2*time.Minute,
			exp: expected{
				said: "Rustpm Test Server will be offline in 2 hours and 1 minute. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
		{
			name:       "2 hours and 30 minutes",
			serverName: "Rustpm Test Server",
			duration:   2*time.Hour + 31*time.Minute,
			exp: expected{
				said: "Rustpm Test Server will be offline in 2 hours and 30 minutes. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			controller := &Controller{
				logger: zap.NewNop(),
			}

			hub := rcon.NewHubMock()
			client, err := hub.Dial(ctx, "test-ip", "test-password")
			require.Nil(t, err)

			when := time.Now().Add(test.duration)

			event := model.Event{
				Schedule: fmt.Sprintf("%d %d * * *", when.Minute(), when.Hour()),
				Kind:     model.EventKindStop,
			}

			server := model.LiveServer{
				Server: model.Server{
					Name: test.serverName,
					Events: []model.Event{
						event,
					},
				},
			}

			err = controller.SayServerTimeRemaining(ctx, server, client)
			require.Nil(t, err)

			rcon, ok := client.(*rcon.ClientMock)
			require.True(t, ok, "expected rcon client to be type *rcon.ClientMock")

			said, err := rcon.Said(ctx)
			require.Nil(t, err)

			require.Equal(t, test.exp.said, said)
		})
	}
}

// --- mocks ---

func newUpdaterMock(expected interface{}) *updaterMock {
	return &updaterMock{
		expected: expected,
	}
}

type updaterMock struct {
	expected interface{}
}

var (
	errUnexpectedType    = errors.New("unexpected type")
	errUnexpectedChanges = errors.New("unexpected changes")
)

func (m updaterMock) Update(ctx context.Context, u gorm.Updater, changes interface{}) error {
	if _, ok := u.(*model.LiveServer); !ok {
		return fmt.Errorf("while checking gorm.Updater type: %w", errUnexpectedType)
	}

	if !reflect.DeepEqual(m.expected, changes) {
		return fmt.Errorf("while checking actual equals expected changes: %w", errUnexpectedChanges)
	}

	return nil
}

func newFinderMock(servers model.LiveServers) *finderMock {
	return &finderMock{servers: servers}
}

type finderMock struct {
	servers model.LiveServers
}

func (m finderMock) Find(ctx context.Context, f gorm.Finder) error {
	servers, ok := f.(*model.LiveServers)
	if !ok {
		return fmt.Errorf("while checking gorm.Finder type: %w", errUnexpectedType)
	}
	*servers = m.servers
	return nil
}

// --- helpers ---

func newLiveServers(cnt int) model.LiveServers {
	var servers model.LiveServers
	for i := 0; i < cnt; i++ {
		// Note: An incomplete representation of a LiveServer is used here because
		// then entirety of a LiveServer is not needed and would only bloat this
		// test.
		server := model.LiveServer{
			Model: imodel.Model{
				ID: uuid.New(),
			},
		}
		servers = append(servers, server)
	}

	return servers
}
