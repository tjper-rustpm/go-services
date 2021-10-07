// +build integration

package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/config"
	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/db/model"
	graphmodel "github.com/tjper/rustcron/cmd/cronman/graph/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var golden = flag.Bool("golden", false, "enable golden tests to overwrite .golden files")

func TestIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s := new(suite)
	s.setup(ctx, t)

	t.Run("create definitions", func(t *testing.T) {
		createDefinitions(ctx, t, s)
	})
	t.Run("get definitions", func(t *testing.T) {
		getDefinitions(ctx, t, s)
	})
	t.Run("update server definitions", func(t *testing.T) {
		updateServerDefinitions(ctx, t, s)
	})
	t.Run("create dormant servers", func(t *testing.T) {
		createDormantServers(ctx, t, s)
	})
	t.Run("create definition moderator", func(t *testing.T) {
		createDefinitionModerator(ctx, t, s)
	})
	t.Run("list moderators pending removal", func(t *testing.T) {
		listModeratorsPendingRemoval(ctx, t, s)
	})
	t.Run("list dormant servers' events", func(t *testing.T) {
		listActiveServerEvents(ctx, t, s)
	})
	t.Run("list created dormant servers", func(t *testing.T) {
		listDormantServers(ctx, t, s)
	})
	t.Run("make servers live", func(t *testing.T) {
		makeServersLive(ctx, t, s)
	})
	t.Run("list live servers' events", func(t *testing.T) {
		listActiveServerEvents(ctx, t, s)
	})
	t.Run("list live servers", func(t *testing.T) {
		listLiveServers(ctx, t, s)
	})
	t.Run("make servers dormant", func(t *testing.T) {
		makeServersDormant(ctx, t, s)
	})
	t.Run("list dormant servers", func(t *testing.T) {
		listDormantServers(ctx, t, s)
	})
	t.Run("make servers archived", func(t *testing.T) {
		makeServersArchived(ctx, t, s)
	})
	t.Run("list no server events", func(t *testing.T) {
		listActiveServerEvents(ctx, t, s)
	})
	t.Run("list archived servers", func(t *testing.T) {
		listArchivedServers(ctx, t, s)
	})
}

func createDefinitions(ctx context.Context, t *testing.T, s *suite) {
	tests := []struct {
		definitionRef **model.ServerDefinition
		definition    model.ServerDefinition
	}{
		0: {
			definitionRef: &s.alphaDefinition,
			definition: model.ServerDefinition{
				Name:                   "Alpha US East",
				InstanceID:             "i-l3n2l5m6h1l5k6",
				InstanceKind:           graphmodel.InstanceKindStandard,
				AllocationID:           "i-d9dk3l20d9n8v7",
				ElasticIP:              "184.83.291.47",
				MaxPlayers:             200,
				MapSize:                uint16(3500),
				MapSeed:                uint16(2000),
				MapSalt:                uint16(3239),
				TickRate:               uint8(30),
				RconPassword:           "alpha-rcon-password",
				Description:            "Alpha US East Vanilla w/ active admins and high-performance and reliable servers.",
				Background:             graphmodel.BackgroundKindForest,
				Url:                    "rustpm.com/alpha-us-east",
				BannerUrl:              "rustpm.com/alpha-us-east/banner",
				WipeDay:                graphmodel.WipeDayFriday,
				BlueprintWipeFrequency: graphmodel.WipeFrequencyMonthly,
				MapWipeFrequency:       graphmodel.WipeFrequencyWeekly,
				Region:                 graphmodel.RegionUsEast,
				Tags: []model.DefinitionTag{
					{Description: "Blueprint Wipe", Icon: graphmodel.IconKindCalendarDay, Value: "Monthly"},
				},
				Events: []model.DefinitionEvent{
					{Day: 1, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 2, Hour: 21, EventKind: graphmodel.EventKindStop},
					{Day: 2, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 3, Hour: 21, EventKind: graphmodel.EventKindStop},
					{Day: 3, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 4, Hour: 21, EventKind: graphmodel.EventKindStop},
					{Day: 4, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 5, Hour: 21, EventKind: graphmodel.EventKindStop},
					{Day: 5, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 6, Hour: 21, EventKind: graphmodel.EventKindStop},
					{Day: 6, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 7, Hour: 21, EventKind: graphmodel.EventKindStop},
					{Day: 7, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 1, Hour: 21, EventKind: graphmodel.EventKindStop},
				},
				Moderators: []model.DefinitionModerator{
					{SteamID: "76561197962911631"},
				},
			},
		},
		1: {
			definitionRef: &s.bravoDefinition,
			definition: model.ServerDefinition{
				Name:                   "Bravo US West",
				InstanceID:             "i-l3n7c5m6h1l5k6",
				InstanceKind:           graphmodel.InstanceKindStandard,
				AllocationID:           "i-d9dk30c0d9n8v7",
				ElasticIP:              "184.83.291.64",
				MaxPlayers:             201,
				MapSize:                uint16(3501),
				MapSeed:                uint16(2001),
				MapSalt:                uint16(3240),
				TickRate:               uint8(30),
				RconPassword:           "bravo-rcon-password",
				Description:            "Bravo US West Vanilla w/ active admins and high-performance and reliable servers.",
				Background:             graphmodel.BackgroundKindJunkyard,
				Url:                    "rustpm.com/bravo-us-east",
				BannerUrl:              "rustpm.com/bravo-us-east/banner",
				WipeDay:                graphmodel.WipeDayFriday,
				BlueprintWipeFrequency: graphmodel.WipeFrequencyMonthly,
				MapWipeFrequency:       graphmodel.WipeFrequencyWeekly,
				Region:                 graphmodel.RegionUsEast,
				Tags: []model.DefinitionTag{
					{Description: "Blueprint Wipe", Icon: graphmodel.IconKindCalendarDay, Value: "Monthly"},
				},
				Events: []model.DefinitionEvent{
					{Day: 1, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 2, Hour: 21, EventKind: graphmodel.EventKindStop},
					{Day: 2, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 3, Hour: 21, EventKind: graphmodel.EventKindStop},
					{Day: 3, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 4, Hour: 21, EventKind: graphmodel.EventKindStop},
					{Day: 4, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 5, Hour: 21, EventKind: graphmodel.EventKindStop},
					{Day: 5, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 6, Hour: 21, EventKind: graphmodel.EventKindStop},
					{Day: 6, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 7, Hour: 21, EventKind: graphmodel.EventKindStop},
					{Day: 7, Hour: 18, EventKind: graphmodel.EventKindStart},
					{Day: 1, Hour: 21, EventKind: graphmodel.EventKindStop},
				},
				Moderators: []model.DefinitionModerator{
					{SteamID: "76561197962911631"},
				},
			},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res, err := s.store.CreateDefinition(ctx, test.definition)
			assert.Nil(t, err)

			*test.definitionRef = res
			clone := res.Clone()
			s.golden(t, clone)
		})
	}
}

func getDefinitions(ctx context.Context, t *testing.T, s *suite) {
	tests := []struct {
		definitionRef **model.ServerDefinition
		id            uuid.UUID
	}{
		0: {
			definitionRef: &s.alphaDefinition,
			id:            s.alphaDefinition.ID,
		},
		1: {
			definitionRef: &s.bravoDefinition,
			id:            s.bravoDefinition.ID,
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res, err := s.store.GetDefinition(ctx, test.id)
			assert.Nil(t, err)

			*test.definitionRef = res
			clone := res.Clone()
			s.golden(t, clone)
		})
	}
}

func updateServerDefinitions(ctx context.Context, t *testing.T, s *suite) {
	tests := []struct {
		definitionRef **model.ServerDefinition
		id            uuid.UUID
		changes       map[string]interface{}
	}{
		0: {
			definitionRef: &s.alphaDefinition,
			id:            s.alphaDefinition.ID,
			changes: map[string]interface{}{
				"maxPlayers": 350,
			},
		},
		1: {
			definitionRef: &s.alphaDefinition,
			id:            s.alphaDefinition.ID,
			changes: map[string]interface{}{
				"name":                   "Omega US East",
				"maxPlayers":             350,
				"mapSize":                4500,
				"mapSeed":                3000,
				"mapSalt":                1000,
				"tickRate":               40,
				"rconPassword":           "omega-rcon-password",
				"description":            "Omega US East w/ active admins, high performance and reliable servers, and high player counts.",
				"url":                    "rustpm.com/omega-us-east",
				"background":             model.BackgroundKindAirport,
				"bannerURL":              "rustpm.com/omega-us-east/banner",
				"wipeDay":                model.WipeDaySaturday,
				"blueprintWipeFrequency": model.WipeFrequencyBiWeekly,
				"mapWipeFrequency":       model.WipeFrequencyBiWeekly,
			},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res, err := s.store.UpdateServerDefinition(ctx, test.id, test.changes)
			assert.Nil(t, err)

			*test.definitionRef = res
			clone := res.Clone()
			s.golden(t, clone)
		})
	}
}

func createDormantServers(ctx context.Context, t *testing.T, s *suite) {
	tests := []struct {
		dormantServerRef **model.DormantServer
		id               uuid.UUID
	}{
		0: {
			dormantServerRef: &s.alphaDormantServer,
			id:               s.alphaDefinition.ID,
		},
		1: {
			dormantServerRef: &s.bravoDormantServer,
			id:               s.bravoDefinition.ID,
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			dormant := model.DormantServer{
				ServerDefinitionID: test.id,
			}
			err := s.store.Create(ctx, &dormant)
			assert.Nil(t, err)

			res, err := s.store.GetDormantServer(ctx, dormant.ID)
			assert.Nil(t, err)

			*test.dormantServerRef = res
			clone := res.Clone()
			s.golden(t, &clone)
		})
	}
}

func createDefinitionModerator(ctx context.Context, t *testing.T, s *suite) {
	tests := []struct {
		dormantServerRef **model.DormantServer
		id               uuid.UUID
		steamID          string
		queuedDeletionAt sql.NullTime
	}{
		0: {
			dormantServerRef: &s.alphaDormantServer,
			id:               s.alphaDefinition.ID,
			steamID:          "moderator-steam-id",
			queuedDeletionAt: sql.NullTime{Time: time.Now(), Valid: true},
		},
		1: {
			dormantServerRef: &s.bravoDormantServer,
			id:               s.bravoDefinition.ID,
			steamID:          "moderator-steam-id",
			queuedDeletionAt: sql.NullTime{},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			err := s.store.Create(
				ctx,
				&model.DefinitionModerator{
					SteamID:            test.steamID,
					QueuedDeletionAt:   test.queuedDeletionAt,
					ServerDefinitionID: test.id,
				},
			)
			assert.Nil(t, err)

			res, err := s.store.GetDormantServer(ctx, (*test.dormantServerRef).ID)
			assert.Nil(t, err)

			*test.dormantServerRef = res
			clone := res.Clone()
			s.golden(t, &clone)
		})
	}
}

func listModeratorsPendingRemoval(ctx context.Context, t *testing.T, s *suite) {
	tests := []struct {
		dormantServerRef **model.DormantServer
		id               uuid.UUID
	}{
		0: {
			dormantServerRef: &s.alphaDormantServer,
			id:               s.alphaDefinition.ID,
		},
		1: {
			dormantServerRef: &s.bravoDormantServer,
			id:               s.bravoDefinition.ID,
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res, err := s.store.ListModeratorsPendingRemoval(ctx, test.id)
			assert.Nil(t, err)

			clone := res.Clone()
			s.golden(t, &clone)
		})
	}
}

func makeServersLive(ctx context.Context, t *testing.T, s *suite) {
	tests := []struct {
		liveServerRef **model.LiveServer
		id            uuid.UUID
	}{
		0: {
			liveServerRef: &s.alphaLiveServer,
			id:            s.alphaDormantServer.ID,
		},
		1: {
			liveServerRef: &s.bravoLiveServer,
			id:            s.bravoDormantServer.ID,
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res, err := s.store.MakeServerLive(
				ctx,
				db.MakeServerLiveInput{
					ID:            test.id,
					AssociationID: "association-ID",
				},
			)
			assert.Nil(t, err)

			*test.liveServerRef = res
			clone := res.Clone()
			s.golden(t, &clone)
		})
	}
}

func makeServersDormant(ctx context.Context, t *testing.T, s *suite) {
	tests := []struct {
		dormantServerRef **model.DormantServer
		id               uuid.UUID
	}{
		0: {
			dormantServerRef: &s.alphaDormantServer,
			id:               s.alphaLiveServer.ID,
		},
		1: {
			dormantServerRef: &s.bravoDormantServer,
			id:               s.bravoLiveServer.ID,
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res, err := s.store.MakeServerDormant(ctx, test.id)
			assert.Nil(t, err)

			*test.dormantServerRef = res
			clone := res.Clone()
			s.golden(t, &clone)
		})
	}
}

func makeServersArchived(ctx context.Context, t *testing.T, s *suite) {
	tests := []struct {
		id uuid.UUID
	}{
		0: {
			id: s.alphaDormantServer.ID,
		},
		1: {
			id: s.bravoDormantServer.ID,
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res, err := s.store.MakeServerArchived(ctx, test.id)
			assert.Nil(t, err)

			clone := res.Clone()
			s.golden(t, &clone)
		})
	}
}

func listLiveServers(ctx context.Context, t *testing.T, s *suite) {
	liveServers := make(model.LiveServers, 0)
	err := s.store.ListServers(ctx, &liveServers)

	assert.Nil(t, err)

	clone := liveServers.Clone()
	s.golden(t, clone)
}

func listDormantServers(ctx context.Context, t *testing.T, s *suite) {
	dormantServers := make(model.DormantServers, 0)
	err := s.store.ListServers(ctx, &dormantServers)
	assert.Nil(t, err)

	clone := dormantServers.Clone()
	s.golden(t, clone)
}

func listArchivedServers(ctx context.Context, t *testing.T, s *suite) {
	archivedServers := make(model.ArchivedServers, 0)
	err := s.store.ListServers(ctx, &archivedServers)
	assert.Nil(t, err)

	clone := archivedServers.Clone()
	s.golden(t, clone)
}

func listActiveServerEvents(ctx context.Context, t *testing.T, s *suite) {
	events, err := s.store.ListActiveServerEvents(ctx)
	assert.Nil(t, err)

	clone := events.Clone()
	s.golden(t, clone)
}

// --- suite ---

type suite struct {
	store db.IStore

	alphaDefinition    *model.ServerDefinition
	alphaDormantServer *model.DormantServer
	alphaLiveServer    *model.LiveServer

	bravoDefinition    *model.ServerDefinition
	bravoDormantServer *model.DormantServer
	bravoLiveServer    *model.LiveServer
}

func (s *suite) setup(ctx context.Context, t *testing.T) {
	cfg := config.Load()

	gorm, err := db.Open(cfg.DSN())
	require.Nil(t, err)

	err = db.Migrate(gorm, cfg.Migrations())
	require.Nil(t, err)

	store := db.NewStore(zap.NewNop(), gorm)
	s.store = store
}

type scrubber interface {
	Scrub()
}

func (s suite) golden(t *testing.T, obj scrubber) {
	obj.Scrub()

	b, err := json.MarshalIndent(obj, "", "\t")
	assert.Nil(t, err)

	assertGolden(t, b)
}

// --- helpers ---

func assertGolden(t *testing.T, b []byte) {
	if *golden {
		err := ioutil.WriteFile(fmt.Sprintf("testdata/%s.golden", t.Name()), b, 0644)
		assert.Nil(t, err)
	}

	exp, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s.golden", t.Name()))
	assert.Nil(t, err)

	assert.JSONEq(t, string(exp), string(b))
}
