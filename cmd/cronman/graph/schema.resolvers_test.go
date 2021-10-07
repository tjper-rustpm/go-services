// +build db-integration

package graph

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/graph/model"
	"github.com/tjper/rustcron/cmd/cronman/migrate"
	"github.com/tjper/rustcron/cmd/cronman/store"

	"github.com/99designs/gqlgen/graphql"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestCreateServer(t *testing.T) {
	type expected struct {
		err    error
		output *model.Server
	}
	tests := []struct {
		input model.NewServer
		exp   expected
	}{
		0: {
			input: model.NewServer{
				Name:        "US East Testing Server",
				Seed:        1,
				MaxPlayers:  200,
				WorldSize:   3123,
				Description: "US East testing server.",
				URL:         "rustpm.com",
				BannerURL:   "rustpm.com",
				Events: []*model.NewEvent{
					{Schedule: "0 0 21 * * *", Type: model.EventTypeLaunch},
					{Schedule: "0 0 9 * * *", Type: model.EventTypeShutdown},
				},
			},
			exp: expected{
				output: &model.Server{
					Name:        "US East Testing Server",
					Seed:        1,
					MaxPlayers:  200,
					WorldSize:   3123,
					Description: "US East testing server.",
					URL:         "rustpm.com",
					BannerURL:   "rustpm.com",
					Events: []*model.Event{
						{Schedule: "0 0 21 * * *", Type: model.EventTypeLaunch},
						{Schedule: "0 0 9 * * *", Type: model.EventTypeShutdown},
					},
				},
			},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			defer upDB(t)(t)
			pool := initDBPool(t, ctx)
			defer pool.Close()

			resolver := NewResolver(zap.NewNop(), store.New(zap.NewNop(), pool))
			ctx = graphql.WithResponseContext(
				ctx,
				graphql.DefaultErrorPresenter,
				graphql.DefaultRecover,
			)

			output, err := resolver.Mutation().CreateServer(ctx, test.input)
			assert.Equal(t, test.exp.err, err)
			assert.True(t, test.exp.output.Equal(*output))
		})
	}
}

func TestArchiveServer(t *testing.T) {
	type expected struct {
		err    error
		output *model.Server
	}
	tests := []struct {
		input model.NewServer
		exp   expected
	}{
		0: {
			input: model.NewServer{
				Name:        "US East Testing Server",
				Seed:        1,
				MaxPlayers:  200,
				WorldSize:   3123,
				Description: "US East testing server.",
				URL:         "rustpm.com",
				BannerURL:   "rustpm.com",
				Events: []*model.NewEvent{
					{Schedule: "0 0 21 * * *", Type: model.EventTypeLaunch},
					{Schedule: "0 0 9 * * *", Type: model.EventTypeShutdown},
				},
			},
			exp: expected{
				output: &model.Server{
					Name:        "US East Testing Server",
					Seed:        1,
					MaxPlayers:  200,
					WorldSize:   3123,
					Description: "US East testing server.",
					URL:         "rustpm.com",
					BannerURL:   "rustpm.com",
					Events: []*model.Event{
						{Schedule: "0 0 21 * * *", Type: model.EventTypeLaunch},
						{Schedule: "0 0 9 * * *", Type: model.EventTypeShutdown},
					},
				},
			},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			defer upDB(t)(t)
			pool := initDBPool(t, ctx)
			defer pool.Close()

			resolver := NewResolver(zap.NewNop(), store.New(zap.NewNop(), pool))
			ctx = graphql.WithResponseContext(
				ctx,
				graphql.DefaultErrorPresenter,
				graphql.DefaultRecover,
			)

			server, err := resolver.Mutation().CreateServer(ctx, test.input)
			assert.Nil(t, err)

			output, err := resolver.Mutation().ArchiveServer(ctx, server.ID)
			assert.Equal(t, test.exp.err, err)
			assert.True(t, test.exp.output.Equal(*output))
		})
	}
}

func TestServers(t *testing.T) {
	type expected struct {
		err    error
		output []*model.Server
	}
	tests := []struct {
		servers  []model.NewServer
		criteria model.ServersCriteria
		exp      expected
	}{
		0: {
			servers: []model.NewServer{
				{
					Name:        "US East Testing Server",
					Seed:        1,
					MaxPlayers:  200,
					WorldSize:   3123,
					Description: "US East testing server.",
					URL:         "rustpm.com",
					BannerURL:   "rustpm.com",
					Events: []*model.NewEvent{
						{Schedule: "0 0 21 * * *", Type: model.EventTypeLaunch},
						{Schedule: "0 0 9 * * *", Type: model.EventTypeShutdown},
					},
				},
				{
					Name:        "US West Testing Server",
					Seed:        2,
					MaxPlayers:  300,
					WorldSize:   3200,
					Description: "US West testing server.",
					URL:         "rustpm.com",
					BannerURL:   "rustpm.com",
					Events: []*model.NewEvent{
						{Schedule: "0 0 23 * * *", Type: model.EventTypeLaunch},
						{Schedule: "0 0 11 * * *", Type: model.EventTypeShutdown},
					},
				},
				{
					Name:        "EU East Testing Server",
					Seed:        3,
					MaxPlayers:  300,
					WorldSize:   3200,
					Description: "EU East testing server.",
					URL:         "rustpm.com",
					BannerURL:   "rustpm.com",
					Events: []*model.NewEvent{
						{Schedule: "0 0 15 * * *", Type: model.EventTypeLaunch},
						{Schedule: "0 0 3 * * *", Type: model.EventTypeShutdown},
					},
				},
			},
			criteria: model.ServersCriteria{
				State:  model.StateActive,
				Limit:  10,
				Offset: 0,
				Order:  model.OrderCreationDesc,
			},
			exp: expected{
				output: []*model.Server{
					{
						Name:        "US East Testing Server",
						Seed:        1,
						MaxPlayers:  200,
						WorldSize:   3123,
						Description: "US East testing server.",
						URL:         "rustpm.com",
						BannerURL:   "rustpm.com",
						Events: []*model.Event{
							{Schedule: "0 0 21 * * *", Type: model.EventTypeLaunch},
							{Schedule: "0 0 9 * * *", Type: model.EventTypeShutdown},
						},
					},
					{
						Name:        "US West Testing Server",
						Seed:        2,
						MaxPlayers:  300,
						WorldSize:   3200,
						Description: "US West testing server.",
						URL:         "rustpm.com",
						BannerURL:   "rustpm.com",
						Events: []*model.Event{
							{Schedule: "0 0 23 * * *", Type: model.EventTypeLaunch},
							{Schedule: "0 0 11 * * *", Type: model.EventTypeShutdown},
						},
					},
					{
						Name:        "EU East Testing Server",
						Seed:        3,
						MaxPlayers:  300,
						WorldSize:   3200,
						Description: "EU East testing server.",
						URL:         "rustpm.com",
						BannerURL:   "rustpm.com",
						Events: []*model.Event{
							{Schedule: "0 0 15 * * *", Type: model.EventTypeLaunch},
							{Schedule: "0 0 3 * * *", Type: model.EventTypeShutdown},
						},
					},
				},
			},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			defer upDB(t)(t)
			pool := initDBPool(t, ctx)
			defer pool.Close()

			resolver := NewResolver(zap.NewNop(), store.New(zap.NewNop(), pool))
			ctx = graphql.WithResponseContext(
				ctx,
				graphql.DefaultErrorPresenter,
				graphql.DefaultRecover,
			)

			// create servers
			for _, server := range test.servers {
				_, err := resolver.Mutation().CreateServer(ctx, server)
				assert.Nil(t, err)
			}

			// retrieve servers
			output, err := resolver.Query().Servers(ctx, test.criteria)
			assert.Equal(t, test.exp.err, err)
			for i := range output {
				assert.True(t, test.exp.output[i].Equal(*output[i]))
			}
		})
	}
}

func TestServer(t *testing.T) {
	type expected struct {
		err    error
		output *model.Server
	}
	tests := []struct {
		input model.NewServer
		exp   expected
	}{
		0: {
			input: model.NewServer{
				Name:        "US East Testing Server",
				Seed:        1,
				MaxPlayers:  200,
				WorldSize:   3123,
				Description: "US East testing server.",
				URL:         "rustpm.com",
				BannerURL:   "rustpm.com",
				Events: []*model.NewEvent{
					{Schedule: "0 0 21 * * *", Type: model.EventTypeLaunch},
					{Schedule: "0 0 9 * * *", Type: model.EventTypeShutdown},
				},
			},
			exp: expected{
				output: &model.Server{
					Name:        "US East Testing Server",
					Seed:        1,
					MaxPlayers:  200,
					WorldSize:   3123,
					Description: "US East testing server.",
					URL:         "rustpm.com",
					BannerURL:   "rustpm.com",
					Events: []*model.Event{
						{Schedule: "0 0 21 * * *", Type: model.EventTypeLaunch},
						{Schedule: "0 0 9 * * *", Type: model.EventTypeShutdown},
					},
				},
			},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			defer upDB(t)(t)
			pool := initDBPool(t, ctx)
			defer pool.Close()

			resolver := NewResolver(zap.NewNop(), store.New(zap.NewNop(), pool))
			ctx = graphql.WithResponseContext(
				ctx,
				graphql.DefaultErrorPresenter,
				graphql.DefaultRecover,
			)

			server, err := resolver.Mutation().CreateServer(ctx, test.input)
			assert.Nil(t, err)

			output, err := resolver.Query().Server(ctx, server.ID)
			assert.Equal(t, test.exp.err, err)
			assert.True(t, test.exp.output.Equal(*output))
		})
	}
}

func TestAddServerEvent(t *testing.T) {
	type expected struct {
		err    error
		output *model.Server
	}
	tests := []struct {
		server model.NewServer
		input  model.NewServerEvent
		exp    expected
	}{
		0: {
			server: model.NewServer{
				Name:        "US East Testing Server",
				Seed:        1,
				MaxPlayers:  200,
				WorldSize:   3123,
				Description: "US East testing server.",
				URL:         "rustpm.com",
				BannerURL:   "rustpm.com",
				Events: []*model.NewEvent{
					{Schedule: "0 0 21 * * *", Type: model.EventTypeLaunch},
					{Schedule: "0 0 9 * * *", Type: model.EventTypeShutdown},
				},
			},
			input: model.NewServerEvent{
				Schedule: "0 0 16 * * *", EventType: model.EventTypeLaunch,
			},
			exp: expected{
				output: &model.Server{
					Name:        "US East Testing Server",
					Seed:        1,
					MaxPlayers:  200,
					WorldSize:   3123,
					Description: "US East testing server.",
					URL:         "rustpm.com",
					BannerURL:   "rustpm.com",
					Events: []*model.Event{
						{Schedule: "0 0 21 * * *", Type: model.EventTypeLaunch},
						{Schedule: "0 0 9 * * *", Type: model.EventTypeShutdown},
						{Schedule: "0 0 16 * * *", Type: model.EventTypeLaunch},
					},
				},
			},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			defer upDB(t)(t)
			pool := initDBPool(t, ctx)
			defer pool.Close()

			resolver := NewResolver(zap.NewNop(), store.New(zap.NewNop(), pool))
			ctx = graphql.WithResponseContext(
				ctx,
				graphql.DefaultErrorPresenter,
				graphql.DefaultRecover,
			)

			server, err := resolver.Mutation().CreateServer(ctx, test.server)
			assert.Nil(t, err)
			test.input.ServerID = server.ID

			output, err := resolver.Mutation().AddServerEvent(ctx, test.input)
			assert.Equal(t, test.exp.err, err)
			assert.True(t, test.exp.output.Equal(*output))
		})
	}
}

func TestRemoveServerEvent(t *testing.T) {
	type expected struct {
		output *model.Server
	}
	tests := []struct {
		server model.NewServer
		input  model.NewServerEvent
		exp    expected
	}{
		0: {
			server: model.NewServer{
				Name:        "US East Testing Server",
				Seed:        1,
				MaxPlayers:  200,
				WorldSize:   3123,
				Description: "US East testing server.",
				URL:         "rustpm.com",
				BannerURL:   "rustpm.com",
				Events: []*model.NewEvent{
					{Schedule: "0 0 21 * * *", Type: model.EventTypeLaunch},
					{Schedule: "0 0 9 * * *", Type: model.EventTypeShutdown},
				},
			},
			input: model.NewServerEvent{
				Schedule:  "0 0 16 * * *",
				EventType: model.EventTypeLaunch,
			},
			exp: expected{
				output: &model.Server{
					Name:        "US East Testing Server",
					Seed:        1,
					MaxPlayers:  200,
					WorldSize:   3123,
					Description: "US East testing server.",
					URL:         "rustpm.com",
					BannerURL:   "rustpm.com",
					Events: []*model.Event{
						{Schedule: "0 0 21 * * *", Type: model.EventTypeLaunch},
						{Schedule: "0 0 9 * * *", Type: model.EventTypeShutdown},
					},
				},
			},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			defer upDB(t)(t)
			pool := initDBPool(t, ctx)
			defer pool.Close()

			store := store.New(zap.NewNop(), pool)
			resolver := NewResolver(zap.NewNop(), store)
			ctx = graphql.WithResponseContext(
				ctx,
				graphql.DefaultErrorPresenter,
				graphql.DefaultRecover,
			)

			server, err := resolver.Mutation().CreateServer(ctx, test.server)
			assert.Nil(t, err)
			test.input.ServerID = server.ID

			event, err := store.CreateServerEvent(ctx, test.input)
			assert.Nil(t, err)

			output, err := resolver.Mutation().RemoveServerEvent(ctx, event.ID)
			assert.Nil(t, err)
			assert.True(t, test.exp.output.Equal(*output))

		})
	}
}

// --- helpers ---

func upDB(t *testing.T) func(t *testing.T) {
	migration, err := migrate.New(
		"file://../../../db/migrations",
		"postgres://postgres:password@localhost:5432?sslmode=disable",
	)
	assert.Nil(t, err)
	assert.Nil(t, migration.Up())
	return func(t *testing.T) {
		assert.Nil(t, migration.Down())
		srcErr, dbErr := migration.Close()
		assert.Nil(t, srcErr)
		assert.Nil(t, dbErr)
	}
}

func initDBPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	pool, err := pgxpool.Connect(
		ctx,
		"postgres://postgres:password@localhost:5432?sslmode=disable",
	)
	assert.Nil(t, err)
	return pool
}
