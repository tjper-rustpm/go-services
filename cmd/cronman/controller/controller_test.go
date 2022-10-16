package controller

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/cmd/cronman/server"
	"github.com/tjper/rustcron/internal/gorm"
	imodel "github.com/tjper/rustcron/internal/model"
	itime "github.com/tjper/rustcron/internal/time"

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

			finder := newLiveServersFinderMock(servers)

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
		update db.UpdateLiveServerInfo
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
			server: model.LiveServer{
				Model: imodel.Model{ID: uuid.New()},
			},
			exp: expected{
				update: db.UpdateLiveServerInfo{
					Changes: map[string]interface{}{
						"active_players": 101,
						"queued_players": 5,
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			check := func(entity gorm.Execer) {
				update, ok := entity.(db.UpdateLiveServerInfo)

				require.True(t, ok, "gorm.Execer was not type db.UpdateLiveServerInfo")
				require.Equal(t, test.exp.update, update)
			}
			test.exp.update.LiveServerID = test.server.ID
			execer := newExecerMock(check)

			controller := &Controller{
				logger: zap.NewNop(),
				execer: execer,
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
			duration:   time.Minute,
			exp: expected{
				said: "Rustpm Test Server will be going offline in 1 minute. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
		{
			name:       "30 minutes",
			serverName: "Rustpm Test Server",
			duration:   30 * time.Minute,
			exp: expected{
				said: "Rustpm Test Server will be going offline in 30 minutes. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
		{
			name:       "1 hour",
			serverName: "Rustpm Test Server",
			duration:   time.Hour,
			exp: expected{
				said: "Rustpm Test Server will be going offline in 1 hour. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
		{
			name:       "1 hour and 30 minutes",
			serverName: "Rustpm Test Server",
			duration:   time.Hour + 30*time.Minute,
			exp: expected{
				said: "Rustpm Test Server will be going offline in 1 hour and 30 minutes. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
		{
			name:       "2 hours",
			serverName: "Rustpm Test Server",
			duration:   2 * time.Hour,
			exp: expected{
				said: "Rustpm Test Server will be going offline in 2 hours. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
		{
			name:       "2 hours and 1 minute",
			serverName: "Rustpm Test Server",
			duration:   2*time.Hour + time.Minute,
			exp: expected{
				said: "Rustpm Test Server will be going offline in 2 hours and 1 minute. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
		{
			name:       "2 hours and 30 minutes",
			serverName: "Rustpm Test Server",
			duration:   2*time.Hour + 30*time.Minute,
			exp: expected{
				said: "Rustpm Test Server will be going offline in 2 hours and 30 minutes. Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			now := time.Now()
			timeMock := itime.NewMock(now)
			timeMock.SetUntil(test.duration)

			when := now.Add(test.duration)
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

			controller := &Controller{
				logger: zap.NewNop(),
				time:   timeMock,
			}

			hub := rcon.NewHubMock()
			client, err := hub.Dial(ctx, "test-ip", "test-password")
			require.Nil(t, err)

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

func TestWipeServer(t *testing.T) {
	t.Parallel()

	type expected struct {
		create db.CreateWipe
	}
	tests := []struct {
		name string
		wipe *model.Wipe
		exp  expected
	}{
		{
			name: "wipe map",
			wipe: model.NewMapWipe(100, 200),
			exp: expected{
				create: db.CreateWipe{
					Wipe: model.Wipe{
						Kind:    model.WipeKindMap,
						MapSeed: 100,
						MapSalt: 200,
					},
				},
			},
		},
		{
			name: "wipe full",
			wipe: model.NewFullWipe(300, 400),
			exp: expected{
				create: db.CreateWipe{
					Wipe: model.Wipe{
						Kind:    model.WipeKindFull,
						MapSeed: 300,
						MapSalt: 400,
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			check := func(entity gorm.Execer) {
				create, ok := entity.(db.CreateWipe)

				require.True(t, ok, "gorm.Execer was not type db.CreateWipe")
				require.Equal(t, test.exp.create, create)
			}
			controller := &Controller{
				logger: zap.NewNop(),
				execer: newExecerMock(check),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			id := uuid.New()
			test.exp.create.ServerID = id

			err := controller.WipeServer(ctx, id, *test.wipe)
			require.Nil(t, err)
		})
	}
}

func TestStartServer(t *testing.T) {
	t.Parallel()

	type expected struct {
		userdataREs         []*regexp.Regexp
		negativeUserdataREs []*regexp.Regexp
		wipeApplied         bool
	}
	tests := map[string]struct {
		dormant model.DormantServer
		exp     expected
	}{
		"no wipe to apply": {
			dormant: model.DormantServer{
				Model: imodel.Model{
					ID: uuid.New(),
				},
				Server: model.Server{
					Model:        imodel.Model{ID: uuid.New()},
					Name:         "",
					RconPassword: "",
					MaxPlayers:   0,
					MapSize:      0,
					TickRate:     0,
					BannerURL:    "",
					Description:  "",
					Options:      nil,
					Region:       model.RegionUsEast,
					Moderators: model.Moderators{
						{SteamID: "admin-users-steam-id"},
					},
					Vips: model.Vips{},
					Wipes: model.Wipes{
						{Model: imodel.Model{At: imodel.At{CreatedAt: time.Now().Add(-24 * time.Hour)}}, Kind: model.WipeKindFull, AppliedAt: sql.NullTime{Time: time.Now().Add(-23 * time.Hour), Valid: true}},
					},
				},
			},
			exp: expected{
				negativeUserdataREs: []*regexp.Regexp{
					regexp.MustCompile(`player\\\.blueprints.+\|\sxargs\srm`),
					regexp.MustCompile(`proceduralmap.+\|\sxargs\srm`),
				},
				wipeApplied: false,
			},
		},
		"map-wipe to apply": {
			dormant: model.DormantServer{
				Model: imodel.Model{
					ID: uuid.New(),
				},
				Server: model.Server{
					Model:        imodel.Model{ID: uuid.New()},
					Name:         "",
					RconPassword: "",
					MaxPlayers:   0,
					MapSize:      0,
					TickRate:     0,
					BannerURL:    "",
					Description:  "",
					Options:      nil,
					Region:       model.RegionUsEast,
					Moderators: model.Moderators{
						{SteamID: "admin-users-steam-id"},
					},
					Vips: model.Vips{},
					Wipes: model.Wipes{
						{
							Model:     imodel.Model{At: imodel.At{CreatedAt: time.Now().Add(-24 * time.Hour)}},
							Kind:      model.WipeKindMap,
							AppliedAt: sql.NullTime{Time: time.Time{}, Valid: false},
						},
					},
				},
			},
			exp: expected{
				userdataREs: []*regexp.Regexp{
					regexp.MustCompile(`proceduralmap.+\|\sxargs\srm`),
				},
				negativeUserdataREs: []*regexp.Regexp{
					regexp.MustCompile(`player\\\.blueprints.+\|\sxargs\srm`),
				},
				wipeApplied: true,
			},
		},
		"full-wipe to apply": {
			dormant: model.DormantServer{
				Model: imodel.Model{
					ID: uuid.New(),
				},
				Server: model.Server{
					Model:        imodel.Model{ID: uuid.New()},
					Name:         "",
					RconPassword: "",
					MaxPlayers:   0,
					MapSize:      0,
					TickRate:     0,
					BannerURL:    "",
					Description:  "",
					Options:      nil,
					Region:       model.RegionUsEast,
					Moderators: model.Moderators{
						{SteamID: "admin-users-steam-id"},
					},
					Vips: model.Vips{},
					Wipes: model.Wipes{
						{
							Model:     imodel.Model{At: imodel.At{CreatedAt: time.Now().Add(-24 * time.Hour)}},
							Kind:      model.WipeKindFull,
							AppliedAt: sql.NullTime{Time: time.Time{}, Valid: false},
						},
					},
				},
			},
			exp: expected{
				userdataREs: []*regexp.Regexp{
					regexp.MustCompile(`player\\\.blueprints.+\|\sxargs\srm`),
					regexp.MustCompile(`proceduralmap.+\|\sxargs\srm`),
				},
				wipeApplied: true,
			},
		},
		"one expired vip": {
			dormant: model.DormantServer{
				Model: imodel.Model{
					ID: uuid.New(),
				},
				Server: model.Server{
					Model:        imodel.Model{ID: uuid.New()},
					Name:         "",
					RconPassword: "",
					MaxPlayers:   0,
					MapSize:      0,
					TickRate:     0,
					BannerURL:    "",
					Description:  "",
					Options:      nil,
					Region:       model.RegionUsEast,
					Moderators: model.Moderators{
						{SteamID: "admin-users-steam-id"},
					},
					Vips: model.Vips{
						{SteamID: "expired-vip-steam-id", ExpiresAt: time.Now().Add(-time.Minute)},
					},
					Wipes: model.Wipes{
						{Model: imodel.Model{At: imodel.At{CreatedAt: time.Now().Add(-24 * time.Hour)}}, Kind: model.WipeKindFull, AppliedAt: sql.NullTime{Time: time.Now().Add(-23 * time.Hour), Valid: true}},
					},
				},
			},
			exp: expected{
				negativeUserdataREs: []*regexp.Regexp{
					regexp.MustCompile(`oxide\.usergroup add expired-vip-steam-id vip`),
				},
				wipeApplied: false,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			serverManager := server.NewMockManager()
			serverManager.SetStartInstanceHandler(func(_ context.Context, _ string, userdata string) error {
				for _, regexp := range test.exp.userdataREs {
					require.Regexp(t, regexp, userdata)
				}
				for _, regexp := range test.exp.negativeUserdataREs {
					require.NotRegexp(t, regexp, userdata)
				}
				return nil
			})
			serverManager.SetMakeInstanceAvailableHandler(func(ctx context.Context, s1, s2 string) (*server.AssociationOutput, error) {
				return &server.AssociationOutput{
					AssociateAddressOutput: ec2.AssociateAddressOutput{
						AssociationId: aws.String("started-server-association-id"),
					},
				}, nil
			})

			var wipeApplied bool
			updateWipeAppliedCheck := func(e gorm.Execer) {
				_, ok := e.(db.UpdateWipeApplied)
				require.True(t, ok)
				wipeApplied = true
			}

			controller := &Controller{
				logger: zap.NewNop(),
				waiter: rcon.NewWaiterMock(100 * time.Millisecond),
				finder: newDormantServerFinderMock(test.dormant),
				execer: newExecerMock(updateWipeAppliedCheck),
				serverController: NewServerDirector(
					serverManager,
					serverManager,
					serverManager,
				),
			}

			id := uuid.New()

			_, err := controller.StartServer(ctx, id)
			require.Nil(t, err)

			require.Equal(t, test.exp.wipeApplied, wipeApplied, "wipe not applied as expected")
		})
	}
}

// --- mocks ---

func newExecerMock(check func(gorm.Execer)) *execerMock {
	return &execerMock{
		check: check,
	}
}

type execerMock struct {
	check func(gorm.Execer)
}

var (
	errUnexpectedType = errors.New("unexpected type")
)

func (m execerMock) Exec(_ context.Context, entity gorm.Execer) error {
	m.check(entity)

	return nil
}

func newLiveServersFinderMock(servers model.LiveServers) *liveServersFinderMock {
	return &liveServersFinderMock{servers: servers}
}

type liveServersFinderMock struct {
	servers model.LiveServers
}

func (m liveServersFinderMock) Find(ctx context.Context, f gorm.Finder) error {
	servers, ok := f.(*model.LiveServers)
	if !ok {
		return fmt.Errorf("while checking gorm.Finder type: %w", errUnexpectedType)
	}
	*servers = m.servers
	return nil
}

func newDormantServerFinderMock(dormant model.DormantServer) *dormantServerFinderMock {
	return &dormantServerFinderMock{dormant: dormant}
}

type dormantServerFinderMock struct {
	dormant model.DormantServer
}

func (m dormantServerFinderMock) Find(ctx context.Context, f gorm.Finder) error {
	finder, ok := f.(*db.FindDormantServer)
	if !ok {
		return fmt.Errorf("while checking gorm.Finder: %w", errUnexpectedType)
	}

	finder.Result = m.dormant
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
