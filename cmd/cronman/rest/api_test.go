package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/internal/healthz"
	ihttp "github.com/tjper/rustcron/internal/http"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	// thursday is used to create a reference to the time.Weekday type.
	thursday = time.Thursday
)

func TestCreateServerInputValidation(t *testing.T) {
	t.Parallel()

	events := Events{
		{Schedule: "40 11 * * *", Kind: model.EventKindStart},
		{Schedule: "0 12 * * *", Kind: model.EventKindLive},
		{Schedule: "0 20 * * *", Kind: model.EventKindStop},
		{Schedule: "0 10 8-14 * *", Weekday: &thursday, Kind: model.EventKindMapWipe},
		{Schedule: "0 10 15-22 * *", Weekday: &thursday, Kind: model.EventKindMapWipe},
		{Schedule: "0 10 23-30 * *", Weekday: &thursday, Kind: model.EventKindMapWipe},
		{Schedule: "0 19 1-7 * *", Weekday: &thursday, Kind: model.EventKindFullWipe},
	}

	type expected struct {
		server model.Server
		status int
	}
	tests := map[string]struct {
		req CreateServerBody
		exp expected
	}{
		"valid server": {
			req: CreateServerBody{
				Name:         "a-valid-server-name",
				InstanceKind: model.InstanceKindSmall,
				MaxPlayers:   200,
				MapSize:      3000,
				MapSeed:      1000,
				MapSalt:      2000,
				TickRate:     30,
				RconPassword: "a-valid-rcon-password",
				Description:  "a-valid-description",
				URL:          "https://rustpm.com",
				Background:   model.BackgroundKindForest,
				BannerURL:    "https://rustpm.com/banner",
				Region:       model.RegionUsEast,
				Options: map[string]interface{}{
					"server.tags": "weekly,vanilla,NA",
				},
				Events: events,
				Moderators: Moderators{
					{SteamID: "87672208073022742"},
				},
				Owners: Owners{
					{SteamID: "76561197962911631"},
				},
				Tags: Tags{
					{Description: "1-valid-tag", Icon: model.IconKindCalendarDay, Value: "1-valid-tag-value"},
				},
			},
			exp: expected{
				server: model.Server{
					Name:         "a-valid-server-name",
					InstanceKind: model.InstanceKindSmall,
					MaxPlayers:   200,
					MapSize:      3000,
					TickRate:     30,
					RconPassword: "a-valid-rcon-password",
					Description:  "a-valid-description",
					URL:          "https://rustpm.com",
					Background:   model.BackgroundKindForest,
					BannerURL:    "https://rustpm.com/banner",
					Region:       model.RegionUsEast,
					Options: map[string]interface{}{
						"server.tags": "weekly,vanilla,NA",
					},
					Wipes: model.Wipes{
						{MapSeed: 1000, MapSalt: 2000, Kind: model.WipeKindFull},
					},
					Events: model.Events{
						{Schedule: "40 11 * * *", Kind: model.EventKindStart},
						{Schedule: "0 12 * * *", Kind: model.EventKindLive},
						{Schedule: "0 20 * * *", Kind: model.EventKindStop},
						{Schedule: "0 10 8-14 * *", Weekday: &thursday, Kind: model.EventKindMapWipe},
						{Schedule: "0 10 15-22 * *", Weekday: &thursday, Kind: model.EventKindMapWipe},
						{Schedule: "0 10 23-30 * *", Weekday: &thursday, Kind: model.EventKindMapWipe},
						{Schedule: "0 19 1-7 * *", Weekday: &thursday, Kind: model.EventKindFullWipe},
					},
					Moderators: model.Moderators{
						{SteamID: "87672208073022742"},
					},
					Owners: model.Owners{
						{SteamID: "76561197962911631"},
					},
					Tags: model.Tags{
						{Description: "1-valid-tag", Icon: model.IconKindCalendarDay, Value: "1-valid-tag-value"},
					},
				},
				status: http.StatusAccepted,
			},
		},
		"missing events": {
			req: CreateServerBody{
				Name:         "a-valid-server-name",
				InstanceKind: model.InstanceKindSmall,
				MaxPlayers:   200,
				MapSize:      3000,
				MapSeed:      1000,
				MapSalt:      2000,
				TickRate:     30,
				RconPassword: "a-valid-rcon-password",
				Description:  "a-valid-description",
				URL:          "https://rustpm.com",
				Background:   model.BackgroundKindForest,
				BannerURL:    "https://rustpm.com/banner",
				Region:       model.RegionUsEast,
				Options: map[string]interface{}{
					"server.tags": "weekly,vanilla,NA",
				},
				Events: Events{},
				Moderators: Moderators{
					{SteamID: "87672208073022742"},
				},
				Owners: Owners{
					{SteamID: "76561197962911631"},
				},
				Tags: Tags{
					{Description: "1-valid-tag", Icon: model.IconKindCalendarDay, Value: "1-valid-tag-value"},
				},
			},
			exp: expected{
				status: http.StatusBadRequest,
			},
		},
		"missing owners": {
			req: CreateServerBody{
				Name:         "a-valid-server-name",
				InstanceKind: model.InstanceKindSmall,
				MaxPlayers:   200,
				MapSize:      3000,
				MapSeed:      1000,
				MapSalt:      2000,
				TickRate:     30,
				RconPassword: "a-valid-rcon-password",
				Description:  "a-valid-description",
				URL:          "https://rustpm.com",
				Background:   model.BackgroundKindForest,
				BannerURL:    "https://rustpm.com/banner",
				Region:       model.RegionUsEast,
				Options: map[string]interface{}{
					"server.tags": "weekly,vanilla,NA",
				},
				Events: events,
				Moderators: Moderators{
					{SteamID: "87672208073022742"},
				},
				Owners: Owners{},
				Tags: Tags{
					{Description: "1-valid-tag", Icon: model.IconKindCalendarDay, Value: "1-valid-tag-value"},
				},
			},
			exp: expected{
				status: http.StatusBadRequest,
			},
		},
		"owners and moderators collision": {
			req: CreateServerBody{
				Name:         "a-valid-server-name",
				InstanceKind: model.InstanceKindSmall,
				MaxPlayers:   200,
				MapSize:      3000,
				MapSeed:      1000,
				MapSalt:      2000,
				TickRate:     30,
				RconPassword: "a-valid-rcon-password",
				Description:  "a-valid-description",
				URL:          "https://rustpm.com",
				Background:   model.BackgroundKindForest,
				BannerURL:    "https://rustpm.com/banner",
				Region:       model.RegionUsEast,
				Options: map[string]interface{}{
					"server.tags": "weekly,vanilla,NA",
				},
				Events: events,
				Moderators: Moderators{
					{SteamID: "76561197962911631"},
				},
				Owners: Owners{
					{SteamID: "76561197962911631"},
				},
				Tags: Tags{
					{Description: "1-valid-tag", Icon: model.IconKindCalendarDay, Value: "1-valid-tag-value"},
				},
			},
			exp: expected{
				status: http.StatusBadRequest,
			},
		},
	}

	for name, test := range tests {
		test := test

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// HTTP endpoint being tested returns an accepted (202) status code and
			// continues to process data critical to test. The done channel allows
			// the test to wait for async processing to complete for the test ends.
			done := make(chan struct{})

			controller := NewControllerMock(
				WithCreateServer(func(ctx context.Context, server model.Server) (*model.DormantServer, error) {
					defer func() {
						close(done)
					}()

					test.exp.server.ID = server.ID

					require.Exactly(t, test.exp.server, server, "created server not as expected")
					return &model.DormantServer{
						Server: server,
					}, nil
				}),
			)

			sessionMiddleware := ihttp.NewSessionMiddlewareMock(
				ihttp.WithInjectSessionIntoCtx(ihttp.SkipMiddleware),
				ihttp.WithTouch(ihttp.SkipMiddleware),
				ihttp.WithHasRole(ihttp.SkipHasRoleMiddleware),
			)

			api := NewAPI(
				zap.NewNop(),
				controller,
				sessionMiddleware,
				healthz.NewHTTP(),
			)

			buf := new(bytes.Buffer)
			err := json.NewEncoder(buf).Encode(test.req)
			require.Nil(t, err)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/server", buf)

			api.Mux.ServeHTTP(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			require.Equal(t, test.exp.status, resp.StatusCode)

			if resp.StatusCode == http.StatusAccepted {
				<-done
			}
		})
	}
}
