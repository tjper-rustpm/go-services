//go:build integration
// +build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/internal/gorm"
	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
)

func TestCreateDormantServer(t *testing.T) {
	tests := []struct {
		name   string
		server *model.DormantServer
	}{
		{
			name: "happy path",
			server: &model.DormantServer{
				Server: model.Server{
					Name:         "test server",
					InstanceID:   "test instance id",
					InstanceKind: model.InstanceKindSmall,
					AllocationID: "test allocation id",
					ElasticIP:    "test elastic ip",
					MaxPlayers:   100,
					MapSize:      2000,
					TickRate:     30,
					RconPassword: "test-rcon-password",
					Description:  "test-description",
					Background:   model.BackgroundKindAirport,
					URL:          "test-url",
					BannerURL:    "test-banner-url",
					Region:       model.RegionUsEast,
					Wipes: model.Wipes{
						{MapSeed: 3000, MapSalt: 4000},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbconn, err := Open(dsn)
			require.Nil(t, err)

			err = Migrate(dbconn, migrations)
			require.Nil(t, err)

			store := gorm.NewStore(dbconn)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err = store.Create(ctx, test.server)
			require.Nil(t, err)
		})
	}
}

func TestMakeServerLive(t *testing.T) {
	tests := []struct {
		name   string
		server *model.DormantServer
	}{
		{
			name: "happy path",
			server: &model.DormantServer{
				Server: model.Server{
					Name:         "test server",
					InstanceID:   "test instance id",
					InstanceKind: model.InstanceKindSmall,
					AllocationID: "test allocation id",
					ElasticIP:    "test elastic ip",
					MaxPlayers:   100,
					MapSize:      2000,
					TickRate:     30,
					RconPassword: "test-rcon-password",
					Description:  "test-description",
					Background:   model.BackgroundKindAirport,
					URL:          "test-url",
					BannerURL:    "test-banner-url",
					Region:       model.RegionUsEast,
					Wipes: model.Wipes{
						{MapSeed: 3000, MapSalt: 4000},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbconn, err := Open(dsn)
			require.Nil(t, err)

			err = Migrate(dbconn, migrations)
			require.Nil(t, err)

			store := NewStore(zap.NewNop(), dbconn)
			storev2 := gorm.NewStore(dbconn)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err = storev2.Create(ctx, test.server)
			require.Nil(t, err)

			_, err = store.MakeServerLive(
				ctx,
				MakeServerLiveInput{
					ID:            test.server.Server.ID,
					AssociationID: "test-association-id",
				},
			)
			require.Nil(t, err)
		})
	}
}

func TestMakeServerDormant(t *testing.T) {
	tests := []struct {
		name   string
		server *model.DormantServer
	}{
		{
			name: "happy path",
			server: &model.DormantServer{
				Server: model.Server{
					Name:         "test server",
					InstanceID:   "test instance id",
					InstanceKind: model.InstanceKindSmall,
					AllocationID: "test allocation id",
					ElasticIP:    "test elastic ip",
					MaxPlayers:   100,
					MapSize:      2000,
					TickRate:     30,
					RconPassword: "test-rcon-password",
					Description:  "test-description",
					Background:   model.BackgroundKindAirport,
					URL:          "test-url",
					BannerURL:    "test-banner-url",
					Region:       model.RegionUsEast,
					Wipes: model.Wipes{
						{MapSeed: 3000, MapSalt: 4000},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbconn, err := Open(dsn)
			require.Nil(t, err)

			err = Migrate(dbconn, migrations)
			require.Nil(t, err)

			store := NewStore(zap.NewNop(), dbconn)
			storev2 := gorm.NewStore(dbconn)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err = storev2.Create(ctx, test.server)
			require.Nil(t, err)

			_, err = store.MakeServerLive(
				ctx,
				MakeServerLiveInput{
					ID:            test.server.Server.ID,
					AssociationID: "test-association-id",
				},
			)
			require.Nil(t, err)

			_, err = store.MakeServerDormant(ctx, test.server.Server.ID)
			require.Nil(t, err)

			var servers model.DormantServers
			err = store.ListServers(ctx, &servers)
			require.Nil(t, err)

			var found bool
			for _, server := range servers {
				if server.Server.ID == test.server.Server.ID {
					found = true
				}
			}
			require.True(t, found, "server not found in list of dormant servers")
		})
	}
}

const (
	dsn        = "host=db user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=UTC"
	migrations = "file://../db/migrations"
)
