package controller

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/cmd/cronman/server"
	imodel "github.com/tjper/rustcron/internal/model"
	itime "github.com/tjper/rustcron/internal/time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	dsn        = os.Getenv("CRONMAN_DSN")
	migrations = os.Getenv("CRONMAN_MIGRATIONS")
)

func TestLiveServerRconForEach(t *testing.T) {
	switch {
	case dsn == "":
		t.Skip("CRONMAN_DSN must be set to execute this test.")
	case migrations == "":
		t.Skip("CRONMAN_MIGRATIONS must be set to execute this test.")
	}

	type expected struct {
		servers model.LiveServers
	}
	tests := []struct {
		name           string
		liveServers    model.LiveServers
		dormantServers model.DormantServers
		exp            expected
	}{
		{
			name: "single live server",
			liveServers: model.LiveServers{
				{Server: *alphaServer.Clone(), AssociationID: "association-id-1"},
			},
			exp: expected{
				servers: model.LiveServers{
					{Server: *alphaServer.Clone(), AssociationID: "association-id-1"},
				},
			},
		},
		{
			name: "two live servers",
			liveServers: model.LiveServers{
				{Server: *alphaServer.Clone(), AssociationID: "association-id-1"},
				{Server: *alphaServer.Clone(), AssociationID: "association-id-2"},
			},
			exp: expected{
				servers: model.LiveServers{
					{Server: *alphaServer.Clone(), AssociationID: "association-id-1"},
					{Server: *alphaServer.Clone(), AssociationID: "association-id-2"},
				},
			},
		},
		{
			name: "five live servers",
			liveServers: model.LiveServers{
				{Server: *alphaServer.Clone(), AssociationID: "association-id-1"},
				{Server: *alphaServer.Clone(), AssociationID: "association-id-2"},
				{Server: *alphaServer.Clone(), AssociationID: "association-id-3"},
				{Server: *alphaServer.Clone(), AssociationID: "association-id-4"},
				{Server: *alphaServer.Clone(), AssociationID: "association-id-5"},
			},
			exp: expected{
				servers: model.LiveServers{
					{Server: *alphaServer.Clone(), AssociationID: "association-id-1"},
					{Server: *alphaServer.Clone(), AssociationID: "association-id-2"},
					{Server: *alphaServer.Clone(), AssociationID: "association-id-3"},
					{Server: *alphaServer.Clone(), AssociationID: "association-id-4"},
					{Server: *alphaServer.Clone(), AssociationID: "association-id-5"},
				},
			},
		},
		{
			name: "one live server, one dormant server",
			liveServers: model.LiveServers{
				{Server: *alphaServer.Clone(), AssociationID: "association-id-1"},
			},
			dormantServers: model.DormantServers{
				{Server: *alphaServer.Clone()},
			},
			exp: expected{
				servers: model.LiveServers{
					{Server: *alphaServer.Clone(), AssociationID: "association-id-1"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			store, err := db.Open(dsn)
			require.Nil(t, err)

			err = db.Migrate(store, migrations)
			require.Nil(t, err)

			for _, server := range test.liveServers {
				server := server

				err := store.WithContext(ctx).Create(&server).Error
				require.Nil(t, err)
				defer func() {
					err := store.WithContext(ctx).Delete(&server).Error
					require.Nil(t, err)
				}()
			}
			for _, server := range test.dormantServers {
				server := server

				err := store.WithContext(ctx).Create(&server).Error
				require.Nil(t, err)
				defer func() {
					err := store.WithContext(ctx).Delete(&server).Error
					require.Nil(t, err)
				}()
			}

			controller := &Controller{
				logger: zap.NewNop(),
				store:  store,
				hub:    rcon.NewHubMock(),
			}

			fn := func(_ context.Context, server model.LiveServer, _ rcon.IRcon) error {
				expect := test.exp.servers[0]
				expect.Model = server.Model
				expect.Server.Model = server.Server.Model

				require.Equal(t, expect.Model.ID, server.Model.ID)
				test.exp.servers = test.exp.servers[1:]
				return nil
			}

			err = controller.LiveServerRconForEach(ctx, fn)
			require.Nil(t, err)
		})
	}
}

func TestCaptureServerInfo(t *testing.T) {
	switch {
	case dsn == "":
		t.Skip("CRONMAN_DSN must be set to execute this test.")
	case migrations == "":
		t.Skip("CRONMAN_MIGRATIONS must be set to execute this test.")
	}

	type expected struct {
		liveServer model.LiveServer
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
				AssociationID: "association-id-1",
				Server:        *alphaServer.Clone(),
			},
			exp: expected{
				liveServer: model.LiveServer{
					ActivePlayers: 101,
					QueuedPlayers: 5,
					Server:        *alphaServer.Clone(),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			store, err := db.Open(dsn)
			require.Nil(t, err)

			err = db.Migrate(store, migrations)
			require.Nil(t, err)

			controller := &Controller{
				logger: zap.NewNop(),
				store:  store,
			}

			err = store.WithContext(ctx).Create(&test.server).Error
			require.Nil(t, err)
			defer func() {
				err = store.WithContext(ctx).Delete(&test.server).Error
				require.Nil(t, err)
			}()

			hub := rcon.NewHubMock(rcon.WithServerInfo(test.serverInfo))
			rcon, err := hub.Dial(ctx, "test-ip", "test-password")
			require.Nil(t, err)

			err = controller.CaptureServerInfo(ctx, test.server, rcon)
			require.Nil(t, err)

			liveServer, err := db.GetLiveServer(ctx, store, test.server.Server.ID)
			require.Nil(t, err)

			test.exp.liveServer.Server.StateID = liveServer.Server.StateID
			test.exp.liveServer.Server.StateType = liveServer.Server.StateType

			test.exp.liveServer.Scrub()
			liveServer.Scrub()
			require.Equal(t, test.exp.liveServer, *liveServer)
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
	if dsn == "" {
		t.Skip("CRONMAN_DSN must be set to execute this test.")
	}
	if migrations == "" {
		t.Skip("CRONMAN_MIGRATIONS must be set to execute this test.")
	}

	type expected struct {
		server model.Server
	}
	tests := []struct {
		name   string
		server model.Server
		wipe   *model.Wipe
		exp    expected
	}{
		{
			name:   "wipe map",
			server: *alphaServer.Clone(),
			wipe:   model.NewMapWipe(100, 200),
			exp: expected{
				server: *alphaServer.Clone(),
			},
		},
		{
			name:   "wipe full",
			server: *alphaServer.Clone(),
			wipe:   model.NewFullWipe(300, 400),
			exp: expected{
				server: *alphaServer.Clone(),
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			store, err := db.Open(dsn)
			require.Nil(t, err)

			err = db.Migrate(store, migrations)
			require.Nil(t, err)

			controller := &Controller{
				logger: zap.NewNop(),
				store:  store,
			}

			err = store.WithContext(ctx).Create(&test.server).Error
			require.Nil(t, err)
			defer func() {
				err = store.WithContext(ctx).Delete(&test.server).Error
				require.Nil(t, err)
			}()

			err = controller.WipeServer(ctx, test.server.ID, *test.wipe)
			require.Nil(t, err)

			// Add created wipe to expected server definition to ensure final
			// comparison correctly checks for its creation.
			test.exp.server.Wipes = append(test.exp.server.Wipes, *test.wipe)

			server, err := db.GetServer(ctx, store, test.server.ID)
			require.Nil(t, err)

			test.exp.server.Scrub()
			server.Scrub()

			require.Equal(t, test.exp.server, *server)
		})
	}
}

func TestStartServer(t *testing.T) {
	switch {
	case dsn == "":
		t.Skip("CRONMAN_DSN must be set to execute this test.")
	case migrations == "":
		t.Skip("CRONMAN_MIGRATIONS must be set to execute this test.")
	}

	oneMinuteAgo := time.Now().Add(-time.Minute).Round(time.Millisecond)
	twentyThreeHoursAgo := time.Now().Add(-23 * time.Hour).Round(time.Millisecond)
	oneDayAgo := time.Now().Add(-24 * time.Hour).Round(time.Millisecond)

	type expected struct {
		userdataREs         []*regexp.Regexp
		negativeUserdataREs []*regexp.Regexp
		server              model.DormantServer
	}
	tests := map[string]struct {
		wipes      model.Wipes
		vips       model.Vips
		owners     model.Owners
		moderators model.Moderators
		exp        expected
	}{
		"queuebypass, adminradar, and vanish plugins": {
			wipes: model.Wipes{
				{
					Model:     imodel.Model{At: imodel.At{CreatedAt: oneDayAgo}},
					Kind:      model.WipeKindFull,
					MapSeed:   3000,
					MapSalt:   4000,
					AppliedAt: sql.NullTime{Time: twentyThreeHoursAgo, Valid: true},
				},
			},
			exp: expected{
				userdataREs: []*regexp.Regexp{
					regexp.MustCompile(`umod\.org\/plugins\/BypassQueue\.cs`),
					regexp.MustCompile(`umod\.org\/plugins\/Vanish\.cs`),
					regexp.MustCompile(`umod\.org\/plugins\/AdminRadar\.cs`),
				},
				negativeUserdataREs: []*regexp.Regexp{
					regexp.MustCompile(`player\\\.blueprints.+\|\sxargs\srm`),
					regexp.MustCompile(`proceduralmap.+\|\sxargs\srm`),
				},
				server: model.DormantServer{
					Server: model.Server{
						Name:         "test-server",
						InstanceID:   "instance-id",
						InstanceKind: model.InstanceKindStandard,
						AllocationID: "allocation-ID",
						ElasticIP:    "elastic-IP",
						MaxPlayers:   200,
						MapSize:      2000,
						TickRate:     30,
						RconPassword: "rcon-password",
						Description:  "description",
						Background:   model.BackgroundKindAirport,
						URL:          "https://rustpm.com",
						BannerURL:    "https://rustpm.com",
						Region:       model.RegionUsEast,
						Options:      map[string]interface{}{},
						Moderators:   model.Moderators{},
						Events:       model.Events{},
						Tags:         model.Tags{},
						Vips:         model.Vips{},
						Owners:       model.Owners{},
						Wipes: model.Wipes{
							{
								Model:     imodel.Model{At: imodel.At{CreatedAt: oneDayAgo}},
								Kind:      model.WipeKindFull,
								MapSeed:   3000,
								MapSalt:   4000,
								AppliedAt: sql.NullTime{Time: twentyThreeHoursAgo, Valid: true},
							},
						},
					},
				},
			},
		},
		"no wipe to apply": {
			wipes: model.Wipes{
				{
					Model:     imodel.Model{At: imodel.At{CreatedAt: oneDayAgo}},
					Kind:      model.WipeKindFull,
					MapSeed:   3000,
					MapSalt:   4000,
					AppliedAt: sql.NullTime{Time: twentyThreeHoursAgo, Valid: true},
				},
			},
			exp: expected{
				negativeUserdataREs: []*regexp.Regexp{
					regexp.MustCompile(`player\\\.blueprints.+\|\sxargs\srm`),
					regexp.MustCompile(`proceduralmap.+\|\sxargs\srm`),
				},
				server: model.DormantServer{
					Server: model.Server{
						Name:         "test-server",
						InstanceID:   "instance-id",
						InstanceKind: model.InstanceKindStandard,
						AllocationID: "allocation-ID",
						ElasticIP:    "elastic-IP",
						MaxPlayers:   200,
						MapSize:      2000,
						TickRate:     30,
						RconPassword: "rcon-password",
						Description:  "description",
						Background:   model.BackgroundKindAirport,
						URL:          "https://rustpm.com",
						BannerURL:    "https://rustpm.com",
						Region:       model.RegionUsEast,
						Options:      map[string]interface{}{},
						Moderators:   model.Moderators{},
						Events:       model.Events{},
						Tags:         model.Tags{},
						Vips:         model.Vips{},
						Owners:       model.Owners{},
						Wipes: model.Wipes{
							{
								Model:     imodel.Model{At: imodel.At{CreatedAt: oneDayAgo}},
								Kind:      model.WipeKindFull,
								MapSeed:   3000,
								MapSalt:   4000,
								AppliedAt: sql.NullTime{Time: twentyThreeHoursAgo, Valid: true},
							},
						},
					},
				},
			},
		},
		"map-wipe to apply": {
			wipes: model.Wipes{
				{
					Model:     imodel.Model{At: imodel.At{CreatedAt: time.Now().Add(-24 * time.Hour)}},
					Kind:      model.WipeKindMap,
					MapSeed:   3000,
					MapSalt:   4000,
					AppliedAt: sql.NullTime{Time: time.Time{}, Valid: false},
				},
			},
			exp: expected{
				userdataREs: []*regexp.Regexp{
					regexp.MustCompile(`proceduralmap.+\|\sxargs\srm`),
				},
				negativeUserdataREs: []*regexp.Regexp{
					regexp.MustCompile(`player\\\.blueprints.+\|\sxargs\srm`),
				},
				server: model.DormantServer{
					Server: model.Server{
						Name:         "test-server",
						InstanceID:   "instance-ID",
						InstanceKind: model.InstanceKindStandard,
						AllocationID: "allocation-ID",
						ElasticIP:    "elastic-IP",
						MaxPlayers:   200,
						MapSize:      2000,
						TickRate:     30,
						RconPassword: "rcon-password",
						Description:  "description",
						Background:   model.BackgroundKindAirport,
						URL:          "https://rustpm.com",
						BannerURL:    "https://rustpm.com",
						Region:       model.RegionUsEast,
						Options:      map[string]interface{}{},
						Moderators:   model.Moderators{},
						Events:       model.Events{},
						Tags:         model.Tags{},
						Vips:         model.Vips{},
						Owners:       model.Owners{},
						Wipes: model.Wipes{
							{
								Model:     imodel.Model{At: imodel.At{CreatedAt: oneDayAgo}},
								Kind:      model.WipeKindMap,
								MapSeed:   3000,
								MapSalt:   4000,
								AppliedAt: sql.NullTime{Time: time.Now(), Valid: true},
							},
						},
					},
				},
			},
		},
		"full-wipe to apply": {
			wipes: model.Wipes{
				{
					Model:     imodel.Model{At: imodel.At{CreatedAt: time.Now().Add(-24 * time.Hour)}},
					Kind:      model.WipeKindFull,
					MapSeed:   3000,
					MapSalt:   4000,
					AppliedAt: sql.NullTime{Time: time.Time{}, Valid: false},
				},
			},
			exp: expected{
				userdataREs: []*regexp.Regexp{
					regexp.MustCompile(`player\\\.blueprints.+\|\sxargs\srm`),
					regexp.MustCompile(`proceduralmap.+\|\sxargs\srm`),
				},
				server: model.DormantServer{
					Server: model.Server{
						Name:         "test-server",
						InstanceID:   "instance-ID",
						InstanceKind: model.InstanceKindStandard,
						AllocationID: "allocation-ID",
						ElasticIP:    "elastic-IP",
						MaxPlayers:   200,
						MapSize:      2000,
						TickRate:     30,
						RconPassword: "rcon-password",
						Description:  "description",
						Background:   model.BackgroundKindAirport,
						URL:          "https://rustpm.com",
						BannerURL:    "https://rustpm.com",
						Region:       model.RegionUsEast,
						Options:      map[string]interface{}{},
						Moderators:   model.Moderators{},
						Events:       model.Events{},
						Tags:         model.Tags{},
						Vips:         model.Vips{},
						Owners:       model.Owners{},
						Wipes: model.Wipes{
							{
								Model:     imodel.Model{At: imodel.At{CreatedAt: oneDayAgo}},
								Kind:      model.WipeKindFull,
								MapSeed:   3000,
								MapSalt:   4000,
								AppliedAt: sql.NullTime{Time: time.Now(), Valid: true},
							},
						},
					},
				},
			},
		},
		"one expired vip": {
			vips: model.Vips{
				{SteamID: "expired-vip-steam-id", ExpiresAt: oneMinuteAgo},
			},
			wipes: model.Wipes{
				{
					Model:     imodel.Model{At: imodel.At{CreatedAt: time.Now().Add(-24 * time.Hour)}},
					Kind:      model.WipeKindFull,
					MapSeed:   3000,
					MapSalt:   4000,
					AppliedAt: sql.NullTime{Time: twentyThreeHoursAgo, Valid: true},
				},
			},
			exp: expected{
				negativeUserdataREs: []*regexp.Regexp{
					regexp.MustCompile(`oxide\.usergroup add expired-vip-steam-id vip`),
				},
				server: model.DormantServer{
					Server: model.Server{
						Name:         "test-server",
						InstanceID:   "instance-ID",
						InstanceKind: model.InstanceKindStandard,
						AllocationID: "allocation-ID",
						ElasticIP:    "elastic-IP",
						MaxPlayers:   200,
						MapSize:      2000,
						TickRate:     30,
						RconPassword: "rcon-password",
						Description:  "description",
						Background:   model.BackgroundKindAirport,
						URL:          "https://rustpm.com",
						BannerURL:    "https://rustpm.com",
						Region:       model.RegionUsEast,
						Options:      map[string]interface{}{},
						Moderators:   model.Moderators{},
						Events:       model.Events{},
						Tags:         model.Tags{},
						Vips: model.Vips{
							{SteamID: "expired-vip-steam-id", ExpiresAt: oneMinuteAgo},
						},
						Owners: model.Owners{},
						Wipes: model.Wipes{
							{
								Model:     imodel.Model{At: imodel.At{CreatedAt: time.Now().Add(-24 * time.Hour)}},
								Kind:      model.WipeKindFull,
								MapSeed:   3000,
								MapSalt:   4000,
								AppliedAt: sql.NullTime{Time: twentyThreeHoursAgo, Valid: true},
							},
						},
					},
				},
			},
		},
		"owner and moderator exist": {
			moderators: model.Moderators{
				{SteamID: "moderator-steam-id"},
			},
			owners: model.Owners{
				{SteamID: "owner-steam-id"},
			},
			wipes: model.Wipes{
				{
					Model:     imodel.Model{At: imodel.At{CreatedAt: oneDayAgo}},
					Kind:      model.WipeKindFull,
					MapSeed:   3000,
					MapSalt:   4000,
					AppliedAt: sql.NullTime{Time: twentyThreeHoursAgo, Valid: true},
				},
			},
			exp: expected{
				userdataREs: []*regexp.Regexp{
					regexp.MustCompile(`moderatorid moderator-steam-id`),
					regexp.MustCompile(`ownerid owner-steam-id`),
				},
				server: model.DormantServer{
					Server: model.Server{
						Name:         "test-server",
						InstanceID:   "instance-ID",
						InstanceKind: model.InstanceKindStandard,
						AllocationID: "allocation-ID",
						ElasticIP:    "elastic-IP",
						MaxPlayers:   200,
						MapSize:      2000,
						TickRate:     30,
						RconPassword: "rcon-password",
						Description:  "description",
						Background:   model.BackgroundKindAirport,
						URL:          "https://rustpm.com",
						BannerURL:    "https://rustpm.com",
						Region:       model.RegionUsEast,
						Options:      map[string]interface{}{},
						Moderators: model.Moderators{
							{SteamID: "moderator-steam-id"},
						},
						Owners: model.Owners{
							{SteamID: "owner-steam-id"},
						},
						Events: model.Events{},
						Tags:   model.Tags{},
						Vips:   model.Vips{},
						Wipes: model.Wipes{
							{
								Model:     imodel.Model{At: imodel.At{CreatedAt: time.Now().Add(-24 * time.Hour)}},
								Kind:      model.WipeKindFull,
								MapSeed:   3000,
								MapSalt:   4000,
								AppliedAt: sql.NullTime{Time: twentyThreeHoursAgo, Valid: true},
							},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			store, err := db.Open(dsn)
			require.Nil(t, err)

			err = db.Migrate(store, migrations)
			require.Nil(t, err)

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

			server := model.DormantServer{Server: *zeroServer.Clone()}
			server.Server.Wipes = test.wipes
			server.Server.Vips = test.vips
			server.Server.Owners = test.owners
			server.Server.Moderators = test.moderators

			err = store.WithContext(ctx).Create(&server).Error
			require.Nil(t, err)
			defer func() {
				err = store.WithContext(ctx).Delete(&server).Error
				require.Nil(t, err)
			}()

			controller := &Controller{
				logger: zap.NewNop(),
				waiter: rcon.NewWaiterMock(100 * time.Millisecond),
				store:  store,
				serverController: NewServerDirector(
					serverManager,
					serverManager,
					serverManager,
				),
			}

			startedServer, err := controller.StartServer(ctx, server.Server.ID)
			require.Nil(t, err)

			for i, wipe := range startedServer.Server.Wipes {
				expected := test.exp.server.Server.Wipes[i].AppliedAt
				actual := wipe.AppliedAt
				require.WithinDuration(t, expected.Time, actual.Time, time.Second)

				test.exp.server.Server.Wipes[i].AppliedAt = actual
			}

			// Set expected server Fields that could not be known ahead of test.
			test.exp.server.Server.StateID = startedServer.Server.StateID
			test.exp.server.Server.StateType = startedServer.Server.StateType

			// Scrub both expected and actual to ensure that non-deterministic fields
			// do not affect equality comparisons.
			startedServer.Scrub()
			test.exp.server.Scrub()

			require.Equal(t, test.exp.server, *startedServer)
		})
	}
}

// alphaServer is a generic server definition that is used by multiple tests.
// Before updating please review the affected tests.
var alphaServer = model.Server{
	Name:         "test-server",
	InstanceID:   "instance-ID",
	InstanceKind: model.InstanceKindStandard,
	AllocationID: "allocation-ID",
	ElasticIP:    "elastic-IP",
	MaxPlayers:   200,
	MapSize:      2000,
	TickRate:     30,
	RconPassword: "rcon-password",
	Description:  "description",
	Background:   model.BackgroundKindAirport,
	URL:          "https://rustpm.com",
	BannerURL:    "https://rustpm.com",
	Region:       model.RegionUsEast,
	Options:      map[string]interface{}{},
	Wipes: model.Wipes{
		{Kind: model.WipeKindFull, MapSeed: 3000, MapSalt: 4000},
	},
	Events: model.Events{
		{Kind: model.EventKindStart, Schedule: "40 11 * * *"},
		{Kind: model.EventKindLive, Schedule: "0 12 * * *"},
		{Kind: model.EventKindStop, Schedule: "0 23 * *"},
	},
	Owners: model.Owners{
		{SteamID: "76561197962911631"},
	},
	Moderators: model.Moderators{},
	Tags:       model.Tags{},
	Vips:       model.Vips{},
}

// zeroServer is a generic server definition that has the fewest additions
// possible. Owners, events, moderators, etc are not defined.
var zeroServer = model.Server{
	Name:         "test-server",
	InstanceID:   "instance-ID",
	InstanceKind: model.InstanceKindStandard,
	AllocationID: "allocation-ID",
	ElasticIP:    "elastic-IP",
	MaxPlayers:   200,
	MapSize:      2000,
	TickRate:     30,
	RconPassword: "rcon-password",
	Description:  "description",
	Background:   model.BackgroundKindAirport,
	URL:          "https://rustpm.com",
	BannerURL:    "https://rustpm.com",
	Region:       model.RegionUsEast,
	Options:      map[string]interface{}{},
	Wipes:        model.Wipes{},
	Events:       model.Events{},
	Owners:       model.Owners{},
	Moderators:   model.Moderators{},
	Tags:         model.Tags{},
	Vips:         model.Vips{},
}
