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
					"activePlayers": 101,
					"queuedPlayers": 5,
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
