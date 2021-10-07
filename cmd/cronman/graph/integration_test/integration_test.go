// +build integration

package integration

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/config"
	"github.com/tjper/rustcron/cmd/cronman/controller"
	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/graph"
	graphmodel "github.com/tjper/rustcron/cmd/cronman/graph/model"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/cmd/cronman/redis"
	"github.com/tjper/rustcron/cmd/cronman/server"
	"gorm.io/gorm"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	redisv8 "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var golden = flag.Bool("golden", false, "enable golden tests to overwrite .golden files")

func TestIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	s := new(suite)
	s.setup(ctx, t)

	t.Run("create server", func(t *testing.T) {
		createServer(ctx, t, s)
	})
	t.Run("get dormant server", func(t *testing.T) {
		servers(ctx, t, s)
	})
	t.Run("update server", func(t *testing.T) {
		updateServer(ctx, t, s)
	})
	t.Run("add server moderators", func(t *testing.T) {
		addServerModerators(ctx, t, s) // Added same moderator
	})
	t.Run("remove server moderators", func(t *testing.T) {
		removeServerModerators(ctx, t, s) // Did not remove moderator
	})
	t.Run("add server tags", func(t *testing.T) {
		addServerTags(ctx, t, s)
	})
	t.Run("remove server tags", func(t *testing.T) {
		removeServerTags(ctx, t, s)
	})
	t.Run("add server events", func(t *testing.T) {
		addServerEvents(ctx, t, s)
	})
	t.Run("remove server events", func(t *testing.T) {
		removeServerEvents(ctx, t, s)
	})
	t.Run("start dormant server", func(t *testing.T) {
		startServer(ctx, t, s)
	})
	t.Run("get live server", func(t *testing.T) {
		servers(ctx, t, s)
	})
	t.Run("stop live server", func(t *testing.T) {
		stopServer(ctx, t, s)
	})
	t.Run("get stopped server", func(t *testing.T) {
		servers(ctx, t, s)
	})
}

func createServer(ctx context.Context, t *testing.T, s *suite) {
	type expected struct {
		err error
	}
	tests := []struct {
		input               graphmodel.NewServer
		serverDefinitionRef **graphmodel.ServerDefinition
		dormantServerRef    **graphmodel.DormantServer
		exp                 expected
	}{
		0: {
			input: graphmodel.NewServer{
				Name:                   "Interation Test Alpha",
				InstanceKind:           graphmodel.InstanceKindStandard,
				MaxPlayers:             200,
				MapSize:                2000,
				MapSeed:                1,
				MapSalt:                1,
				TickRate:               30,
				RconPassword:           "rconpassword",
				Description:            "Rustpm integration test server.",
				URL:                    "rustpm.com",
				Background:             graphmodel.BackgroundKindAirport,
				BannerURL:              "rustpm.com/banner",
				WipeDay:                graphmodel.WipeDayFriday,
				BlueprintWipeFrequency: graphmodel.WipeFrequencyMonthly,
				MapWipeFrequency:       graphmodel.WipeFrequencyWeekly,
				Region:                 graphmodel.RegionUsEast,
				Schedule: []*graphmodel.NewEvent{
					{Day: 1, Hour: 18, Kind: graphmodel.EventKindStart},
					{Day: 2, Hour: 21, Kind: graphmodel.EventKindStop},
					{Day: 2, Hour: 18, Kind: graphmodel.EventKindStart},
					{Day: 3, Hour: 21, Kind: graphmodel.EventKindStop},
					{Day: 3, Hour: 18, Kind: graphmodel.EventKindStart},
					{Day: 4, Hour: 21, Kind: graphmodel.EventKindStop},
					{Day: 4, Hour: 18, Kind: graphmodel.EventKindStart},
					{Day: 5, Hour: 21, Kind: graphmodel.EventKindStop},
					{Day: 5, Hour: 18, Kind: graphmodel.EventKindStart},
					{Day: 6, Hour: 21, Kind: graphmodel.EventKindStop},
					{Day: 6, Hour: 18, Kind: graphmodel.EventKindStart},
					{Day: 7, Hour: 21, Kind: graphmodel.EventKindStop},
					{Day: 7, Hour: 18, Kind: graphmodel.EventKindStart},
					{Day: 1, Hour: 21, Kind: graphmodel.EventKindStop},
				},
				Moderators: []*graphmodel.NewModerator{
					{SteamID: "76561197962911631"},
				},
				Tags: []*graphmodel.NewTag{
					{Description: "Region", Icon: graphmodel.IconKindGlobe, Value: "US East"},
					{Description: "Map Size", Icon: graphmodel.IconKindMap, Value: "4000"},
					{Description: "Map Seed", Icon: graphmodel.IconKindGames, Value: "3253"},
				},
			},
			dormantServerRef:    &s.alphaDormantServer,
			serverDefinitionRef: &s.alphaServerDefinition,
			exp:                 expected{},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output, err := s.resolver.Mutation().CreateServer(ctx, test.input)
			assert.Equal(t, test.exp.err, err)

			*test.dormantServerRef = output.Server
			*test.serverDefinitionRef = output.Server.Definition
			cloned := output.Server.Definition.Clone()

			cloned.Scrub()
			assertGolden(t, cloned)
		})
	}
}

func servers(ctx context.Context, t *testing.T, s *suite) {
	type expected struct {
		err error
	}
	tests := []struct {
		input graphmodel.ServersCriteria
		exp   expected
	}{
		0: {
			input: graphmodel.ServersCriteria{State: graphmodel.StateActive},
			exp:   expected{},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output, err := s.resolver.Query().Servers(ctx, test.input)
			assert.Equal(t, test.exp.err, err)

			for i := range output.Servers {
				server, ok := output.Servers[i].(interface{ Scrub() })
				assert.True(t, ok, "server type scrubber")
				server.Scrub()
			}

			assertGolden(t, output.Servers)
		})
	}
}

func updateServer(ctx context.Context, t *testing.T, s *suite) {
	type expected struct {
		err error
	}
	tests := []struct {
		id                  string
		changes             map[string]interface{}
		serverDefinitionRef **graphmodel.ServerDefinition
		exp                 expected
	}{
		0: {
			id: s.alphaServerDefinition.ID,
			changes: map[string]interface{}{
				"mapSize": 4000,
			},
			serverDefinitionRef: &s.alphaServerDefinition,
			exp:                 expected{},
		},
		1: {
			id: s.alphaServerDefinition.ID,
			changes: map[string]interface{}{
				"name":                   "Integration Test Omega",
				"maxPlayers":             300,
				"mapSize":                1000,
				"mapSeed":                1234,
				"mapSalt":                1234,
				"tickRate":               40,
				"rconPassword":           "omega-rcon-password",
				"description":            "Rustpm Omega integration test server.",
				"url":                    "rustpm.com/omega",
				"background":             graphmodel.BackgroundKindForest,
				"bannerURL":              "rustpm.com/omega/banner",
				"wipeDay":                graphmodel.WipeDayTuesday,
				"blueprintWipeFrequency": graphmodel.WipeFrequencyBiweekly,
				"mapWipeFrequency":       graphmodel.WipeFrequencyBiweekly,
			},
			serverDefinitionRef: &s.alphaServerDefinition,
			exp:                 expected{},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output, err := s.resolver.Mutation().UpdateServer(ctx, test.id, test.changes)
			assert.Equal(t, test.exp.err, err)

			*test.serverDefinitionRef = output.Definition
			cloned := output.Definition.Clone()

			cloned.Scrub()
			assertGolden(t, cloned)
		})
	}
}

func addServerModerators(ctx context.Context, t *testing.T, s *suite) {
	type expected struct {
		err error
	}
	tests := []struct {
		id                  string
		mods                []*graphmodel.NewModerator
		serverDefinitionRef **graphmodel.ServerDefinition
		moderatorsRef       *[]*graphmodel.Moderator
		exp                 expected
	}{
		0: {
			id: s.alphaServerDefinition.ID,
			mods: []*graphmodel.NewModerator{
				{SteamID: "76561197962911764"},
			},
			serverDefinitionRef: &s.alphaServerDefinition,
			moderatorsRef:       &s.moderators,
			exp:                 expected{},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output, err := s.resolver.Mutation().AddServerModerators(ctx, test.id, test.mods)
			assert.Equal(t, test.exp.err, err)

			*test.moderatorsRef = output.Definition.Moderators
			*test.serverDefinitionRef = output.Definition
			cloned := output.Definition.Clone()

			cloned.Scrub()
			assertGolden(t, cloned)
		})
	}
}

func removeServerModerators(ctx context.Context, t *testing.T, s *suite) {
	type expected struct {
		err error
	}
	tests := []struct {
		serverId            string
		moderatorIds        []string
		serverDefinitionRef **graphmodel.ServerDefinition
		exp                 expected
	}{
		0: {
			serverId:            s.alphaServerDefinition.ID,
			moderatorIds:        moderatorIds(s.moderators[1:]),
			serverDefinitionRef: &s.alphaServerDefinition,
			exp:                 expected{},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output, err := s.resolver.Mutation().RemoveServerModerators(ctx, test.serverId, test.moderatorIds)
			assert.Equal(t, test.exp.err, err)

			*test.serverDefinitionRef = output.Definition
			cloned := output.Definition.Clone()

			cloned.Scrub()
			assertGolden(t, cloned)
		})
	}
}

func addServerTags(ctx context.Context, t *testing.T, s *suite) {
	type expected struct {
		err error
	}
	tests := []struct {
		id                  string
		tags                []*graphmodel.NewTag
		serverDefinitionRef **graphmodel.ServerDefinition
		tagsRef             *[]*graphmodel.Tag
		exp                 expected
	}{
		0: {
			id: s.alphaServerDefinition.ID,
			tags: []*graphmodel.NewTag{
				{Description: "Map Wipe Schedule", Icon: graphmodel.IconKindCalendarWeek, Value: "Bi-Weekly"},
			},
			serverDefinitionRef: &s.alphaServerDefinition,
			tagsRef:             &s.tags,
			exp:                 expected{},
		},
		1: {
			id: s.alphaServerDefinition.ID,
			tags: []*graphmodel.NewTag{
				{Description: "Blueprint Wipe Schedule", Icon: graphmodel.IconKindCalendarEvent, Value: "Monthly"},
				{Description: "Game Mode", Icon: graphmodel.IconKindGames, Value: "Vanilla"},
			},
			serverDefinitionRef: &s.alphaServerDefinition,
			tagsRef:             &s.tags,
			exp:                 expected{},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output, err := s.resolver.Mutation().AddServerTags(ctx, test.id, test.tags)
			assert.Equal(t, test.exp.err, err)

			*test.tagsRef = output.Definition.Tags
			*test.serverDefinitionRef = output.Definition
			cloned := output.Definition.Clone()

			cloned.Scrub()
			assertGolden(t, cloned)
		})
	}
}

func removeServerTags(ctx context.Context, t *testing.T, s *suite) {
	type expected struct {
		err error
	}
	tests := []struct {
		serverId            string
		tagIds              []string
		serverDefinitionRef **graphmodel.ServerDefinition
		exp                 expected
	}{
		0: {
			serverId:            s.alphaServerDefinition.ID,
			tagIds:              tagIds(s.tags[3:]),
			serverDefinitionRef: &s.alphaServerDefinition,
			exp:                 expected{},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output, err := s.resolver.Mutation().RemoveServerTags(ctx, test.serverId, test.tagIds)
			assert.Equal(t, test.exp.err, err)

			*test.serverDefinitionRef = output.Definition
			cloned := output.Definition.Clone()
			cloned.Scrub()
			assertGolden(t, cloned)
		})
	}
}

func addServerEvents(ctx context.Context, t *testing.T, s *suite) {
	type expected struct {
		err error
	}
	tests := []struct {
		id                  string
		events              []*graphmodel.NewEvent
		serverDefinitionRef **graphmodel.ServerDefinition
		scheduleRef         *[]*graphmodel.Event
		exp                 expected
	}{
		0: {
			id: s.alphaServerDefinition.ID,
			events: []*graphmodel.NewEvent{
				{Day: 7, Hour: 2, Kind: graphmodel.EventKindStart},
				{Day: 7, Hour: 8, Kind: graphmodel.EventKindStop},
			},
			serverDefinitionRef: &s.alphaServerDefinition,
			scheduleRef:         &s.schedule,
			exp:                 expected{},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output, err := s.resolver.Mutation().AddServerEvents(ctx, test.id, test.events)
			assert.Equal(t, test.exp.err, err)

			*test.scheduleRef = output.Definition.Schedule
			*test.serverDefinitionRef = output.Definition

			cloned := output.Definition.Clone()
			cloned.Scrub()
			assertGolden(t, cloned)
		})
	}
}

func removeServerEvents(ctx context.Context, t *testing.T, s *suite) {
	type expected struct {
		err error
	}
	tests := []struct {
		serverId            string
		eventIds            []string
		serverDefinitionRef **graphmodel.ServerDefinition
		exp                 expected
	}{
		0: {
			serverId:            s.alphaServerDefinition.ID,
			eventIds:            eventIds(s.schedule[10:]),
			serverDefinitionRef: &s.alphaServerDefinition,
			exp:                 expected{},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output, err := s.resolver.Mutation().RemoveServerEvents(ctx, test.serverId, test.eventIds)
			assert.Equal(t, test.exp.err, err)

			*test.serverDefinitionRef = output.Definition
			cloned := output.Definition.Clone()
			cloned.Scrub()
			assertGolden(t, cloned)
		})
	}
}

func startServer(ctx context.Context, t *testing.T, s *suite) {
	type expected struct {
		err error
	}
	tests := []struct {
		dormantServer *graphmodel.DormantServer
		liveServerRef **graphmodel.LiveServer
		exp           expected
	}{
		0: {
			dormantServer: s.alphaDormantServer,
			liveServerRef: &s.alphaLiveServer,
			exp:           expected{},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output, err := s.resolver.Mutation().StartServer(ctx, test.dormantServer.ID)
			assert.Equal(t, test.exp.err, err)

			*test.liveServerRef = output.Server

			cloned := output.Server.Clone()
			cloned.Scrub()
			assertGolden(t, cloned)
		})
	}
}

func stopServer(ctx context.Context, t *testing.T, s *suite) {
	type expected struct {
		err error
	}
	tests := []struct {
		liveServer       *graphmodel.LiveServer
		dormantServerRef **graphmodel.DormantServer
		exp              expected
	}{
		0: {
			liveServer:       s.alphaLiveServer,
			dormantServerRef: &s.alphaDormantServer,
			exp:              expected{},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output, err := s.resolver.Mutation().StopServer(ctx, test.liveServer.ID)
			assert.Equal(t, test.exp.err, err)

			*test.dormantServerRef = output.Server

			cloned := output.Server.Clone()

			cloned.Scrub()
			assertGolden(t, cloned)
		})
	}
}

// --- suite ---
// the suite contains shared components between all tests. It is here that the
// the integration test dependencies are declared, initialized, and shared.

type suite struct {
	resolver *graph.Resolver

	alphaServerDefinition *graphmodel.ServerDefinition
	alphaDormantServer    *graphmodel.DormantServer
	alphaLiveServer       *graphmodel.LiveServer

	moderators []*graphmodel.Moderator
	schedule   []*graphmodel.Event
	tags       []*graphmodel.Tag
}

func (s *suite) setup(ctx context.Context, t *testing.T) {
	cfg := config.Load()
	dbconn := openDb(t, cfg)
	migrateUp(t, dbconn, cfg)
	rdb := dialRedis(ctx, t, cfg)
	serverDirector := newServerDirector(ctx, t)

	ctrl := controller.New(
		zap.NewExample(),
		redis.New(rdb),
		db.NewStore(zap.NewExample(), dbconn),
		serverDirector,
		controller.NewHub(zap.NewExample()),
		rcon.NewWaiter(zap.NewExample()),
		controller.NewNotifier(zap.NewExample(), rdb),
	)

	s.resolver = graph.NewResolver(
		zap.NewExample(),
		ctrl,
	)
}

// --- helpers ---

func migrateUp(t *testing.T, dbconn *gorm.DB, config *config.Config) {
	err := db.Migrate(dbconn, config.Migrations())
	require.Nil(t, err)
}

func openDb(t *testing.T, config *config.Config) *gorm.DB {
	dbconn, err := db.Open(config.DSN())
	require.Nil(t, err)
	return dbconn
}

func dialRedis(ctx context.Context, t *testing.T, config *config.Config) *redisv8.Client {
	rdb := redisv8.NewClient(&redisv8.Options{
		Addr:     config.RedisAddr(),
		Password: config.RedisPassword(),
	})
	err := rdb.Ping(ctx).Err()
	require.Nil(t, err)
	return rdb
}

func newServerDirector(ctx context.Context, t *testing.T) *controller.ServerDirector {
	awscfg, err := awsconfig.LoadDefaultConfig(ctx)
	require.Nil(t, err)

	usEastEC2 := ec2.NewFromConfig(awscfg, func(opts *ec2.Options) {
		opts.Region = "us-east-1"
	})
	usWestEC2 := ec2.NewFromConfig(awscfg, func(opts *ec2.Options) {
		opts.Region = "us-west-1"
	})
	euCentralEC2 := ec2.NewFromConfig(awscfg, func(opts *ec2.Options) {
		opts.Region = "eu-central-1"
	})
	return controller.NewServerDirector(
		server.NewManager(zap.NewExample(), usEastEC2),
		server.NewManager(zap.NewExample(), usWestEC2),
		server.NewManager(zap.NewExample(), euCentralEC2),
	)
}

func moderatorIds(mods []*graphmodel.Moderator) []string {
	ids := make([]string, 0, len(mods))
	for _, mod := range mods {
		ids = append(ids, mod.ID)
	}
	return ids
}

func tagIds(tags []*graphmodel.Tag) []string {
	ids := make([]string, 0, len(tags))
	for _, tag := range tags {
		ids = append(ids, tag.ID)
	}
	return ids
}

func eventIds(events []*graphmodel.Event) []string {
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}
	return ids
}

func assertGolden(t *testing.T, obj interface{}) {
	b, err := json.MarshalIndent(obj, "", "\t")
	assert.Nil(t, err)

	if *golden {
		err := ioutil.WriteFile(fmt.Sprintf("testdata/%s.golden", t.Name()), b, 0644)
		assert.Nil(t, err)
	}

	exp, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s.golden", t.Name()))
	assert.Nil(t, err)

	assert.Equal(t, string(exp), string(b))
}
