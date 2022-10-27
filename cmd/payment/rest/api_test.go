package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/payment/db"
	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/internal/healthz"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/stream"
	"github.com/tjper/rustcron/internal/stripe"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestFindServers(t *testing.T) {
	t.Parallel()

	type expected struct {
		servers model.Servers
		status  int
	}
	tests := map[string]struct {
		servers model.Servers
		exp     expected
	}{
		"no servers": {
			servers: model.Servers{},
			exp: expected{
				servers: model.Servers{},
				status:  http.StatusOK,
			},
		},
		"one server": {
			servers: model.Servers{
				{ActiveSubscriptions: 100, SubscriptionLimit: 200},
			},
			exp: expected{
				servers: model.Servers{
					{ActiveSubscriptions: 100, SubscriptionLimit: 200},
				},
				status: http.StatusOK,
			},
		},
	}

	for name, test := range tests {
		test := test

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			store := db.NewStoreMock(
				db.WithFindServers(func(context.Context) (model.Servers, error) {
					return test.servers, nil
				}),
			)

			sessionMiddleware := ihttp.NewSessionMiddlewareMock(
				ihttp.WithInjectSessionIntoCtx(ihttp.SkipMiddleware),
				ihttp.WithTouch(ihttp.SkipMiddleware),
				ihttp.WithHasRole(ihttp.SkipMiddleware),
				ihttp.WithIsAuthenticated(ihttp.SkipMiddleware),
			)

			api := NewAPI(
				zap.NewNop(),
				store,
				staging.NewClientMock(),
				stream.NewClientMock(),
				stripe.NewMock(),
				sessionMiddleware,
				healthz.NewHTTP(),
			)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/v1/servers", nil)

			api.Mux.ServeHTTP(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			require.Equal(t, test.exp.status, resp.StatusCode)

			if resp.StatusCode != http.StatusOK {
				return
			}

			var servers model.Servers
			err := json.NewDecoder(resp.Body).Decode(&servers)
			require.Nil(t, err)

			require.Equal(t, test.exp.servers, servers)
		})
	}
}

func TestUpdateServer(t *testing.T) {
	t.Parallel()

	alpha := uuid.New()

	type expected struct {
		serverID uuid.UUID
		changes  map[string]interface{}
		resp     *model.Server
		status   int
	}
	tests := map[string]struct {
		req map[string]interface{}
		exp expected
	}{
		"valid server update": {
			req: map[string]interface{}{
				"id": alpha,
				"changes": map[string]interface{}{
					"subscriptionLimit": float64(300),
				},
			},
			exp: expected{
				serverID: alpha,
				changes: map[string]interface{}{
					"subscriptionLimit": float64(300),
				},
				resp: &model.Server{
					ID:                  alpha,
					ActiveSubscriptions: 0,
					SubscriptionLimit:   300,
				},
				status: http.StatusCreated,
			},
		},
		"missing changes": {
			req: map[string]interface{}{
				"id": alpha,
			},
			exp: expected{
				status: http.StatusBadRequest,
			},
		},
		"missing id": {
			req: map[string]interface{}{
				"changes": map[string]interface{}{
					"subscriptionLimit": float64(300),
				},
			},
			exp: expected{
				status: http.StatusBadRequest,
			},
		},
		"invalid changes field": {
			req: map[string]interface{}{
				"id": alpha,
				"changes": map[string]interface{}{
					"invalidField": float64(300),
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

			store := db.NewStoreMock(
				db.WithUpdateServer(func(_ context.Context, serverID uuid.UUID, changes map[string]interface{}) (*model.Server, error) {
					require.Equal(t, test.exp.serverID, serverID)
					require.Equal(t, test.exp.changes, changes)

					return test.exp.resp, nil
				}),
			)

			sessionMiddleware := ihttp.NewSessionMiddlewareMock(
				ihttp.WithInjectSessionIntoCtx(ihttp.SkipMiddleware),
				ihttp.WithTouch(ihttp.SkipMiddleware),
				ihttp.WithHasRole(ihttp.SkipMiddleware),
				ihttp.WithIsAuthenticated(ihttp.SkipMiddleware),
			)

			api := NewAPI(
				zap.NewNop(),
				store,
				staging.NewClientMock(),
				stream.NewClientMock(),
				stripe.NewMock(),
				sessionMiddleware,
				healthz.NewHTTP(),
			)

			buf := new(bytes.Buffer)
			err := json.NewEncoder(buf).Encode(test.req)
			require.Nil(t, err)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, "/v1/server", buf)

			api.Mux.ServeHTTP(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			require.Equal(t, test.exp.status, resp.StatusCode)

			if resp.StatusCode != http.StatusOK {
				return
			}

			var server model.Server
			err = json.NewDecoder(resp.Body).Decode(&server)
			require.Nil(t, err)
		})
	}
}
