package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v72"
	"github.com/tjper/rustcron/cmd/payment/db"
	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/internal/gorm"
	"github.com/tjper/rustcron/internal/healthz"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
	"github.com/tjper/rustcron/internal/stream"
	istripe "github.com/tjper/rustcron/internal/stripe"

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
				istripe.NewMock(),
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
				istripe.NewMock(),
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

func TestBilling(t *testing.T) {
	type shared struct {
		userID           uuid.UUID
		stripeCustomerID string
	}
	type expected struct {
		status   int
		redirect Redirect
	}
	tests := map[string]struct {
		returnURL             string
		billingURL            string
		firstCustomerByUserID func(*testing.T, *shared) func(context.Context, uuid.UUID) (*model.Customer, error)
		exp                   expected
	}{
		"happy path": {
			returnURL:  "https://rustpm.com/servers",
			billingURL: "https://stripe.com/billing",
			firstCustomerByUserID: func(t *testing.T, shared *shared) func(context.Context, uuid.UUID) (*model.Customer, error) {
				return func(_ context.Context, userID uuid.UUID) (*model.Customer, error) {
					require.Equal(t, shared.userID, userID)

					shared.stripeCustomerID = uuid.NewString()
					return &model.Customer{
						StripeCustomerID: shared.stripeCustomerID,
					}, nil
				}
			},
			exp: expected{
				status:   http.StatusCreated,
				redirect: Redirect{URL: "https://stripe.com/billing"},
			},
		},
		"empty returnUrl": {
			returnURL: "",
			firstCustomerByUserID: func(t *testing.T, _ *shared) func(context.Context, uuid.UUID) (*model.Customer, error) {
				return func(_ context.Context, _ uuid.UUID) (*model.Customer, error) {
					require.FailNow(t, "store.FirstCustomerByUserID should not be called.")
					return nil, nil
				}
			},
			exp: expected{
				status: http.StatusBadRequest,
			},
		},
		"customer not found": {
			returnURL: "https://rustpm.com/servers",
			firstCustomerByUserID: func(t *testing.T, shared *shared) func(context.Context, uuid.UUID) (*model.Customer, error) {
				return func(_ context.Context, userID uuid.UUID) (*model.Customer, error) {
					require.Equal(t, shared.userID, userID)
					return nil, gorm.ErrNotFound
				}
			},
			exp: expected{
				status: http.StatusNotFound,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			shared := &shared{}

			store := db.NewStoreMock(
				db.WithFirstCustomerByUserID(test.firstCustomerByUserID(t, shared)),
			)

			stripe := istripe.NewMock(
				istripe.WithBillingPortalSession(
					func(params *stripe.BillingPortalSessionParams) (string, error) {
						require.Equal(t, test.returnURL, *params.ReturnURL)
						require.Equal(t, shared.stripeCustomerID, *params.Customer)
						return test.billingURL, nil
					}),
			)

			injectSession := func(next http.Handler) http.Handler {
				return http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						shared.userID = uuid.New()

						sess := &session.Session{
							ID: uuid.NewString(),
							User: session.User{
								ID: shared.userID,
							},
							AbsoluteExpiration: time.Now().Add(time.Minute).UTC(),
							LastActivityAt:     time.Now().UTC(),
							RefreshedAt:        time.Now().UTC(),
							CreatedAt:          time.Now().UTC(),
						}
						ctx := session.WithSession(r.Context(), sess)
						r = r.WithContext(ctx)

						next.ServeHTTP(w, r)
					},
				)
			}

			sessionMiddleware := ihttp.NewSessionMiddlewareMock(
				ihttp.WithInjectSessionIntoCtx(injectSession),
				ihttp.WithTouch(ihttp.SkipMiddleware),
				ihttp.WithHasRole(ihttp.SkipMiddleware),
				ihttp.WithIsAuthenticated(ihttp.SkipMiddleware),
			)
			api := NewAPI(
				zap.NewNop(),
				store,
				staging.NewClientMock(),
				stream.NewClientMock(),
				stripe,
				sessionMiddleware,
				healthz.NewHTTP(),
			)

			body := map[string]interface{}{"returnUrl": test.returnURL}
			buf := new(bytes.Buffer)
			err := json.NewEncoder(buf).Encode(body)
			require.Nil(t, err)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/billing", buf)

			api.Mux.ServeHTTP(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			require.Equal(t, test.exp.status, resp.StatusCode)

			if resp.StatusCode != http.StatusCreated {
				return
			}

			var redirect Redirect
			err = json.NewDecoder(resp.Body).Decode(&redirect)
			require.Nil(t, err)

			require.Equal(t, test.exp.redirect, redirect)
		})
	}
}

func TestCheckout(t *testing.T) {
	type shared struct {
		serverID          uuid.UUID
		steamID           string
		clientReferenceID string
		expiresAt         time.Time
	}
	type expected struct {
		status   int
		redirect Redirect
	}

	tests := map[string]struct {
		checkoutEnabled      bool
		body                 func(*shared) map[string]interface{}
		firstServerByID      func(*testing.T, *shared) func(context.Context, uuid.UUID) (*model.Server, error)
		isServerVipBySteamID func(*testing.T, *shared) func(context.Context, uuid.UUID, string) (bool, error)
		stageCheckout        func(*testing.T, *shared) func(context.Context, interface{}, time.Time) (string, error)
		checkoutSession      func(*testing.T, *shared) func(*stripe.CheckoutSessionParams) (string, error)
		exp                  expected
	}{
		"checkout disabled": {
			body: func(shared *shared) map[string]interface{} {
				return map[string]interface{}{
					"serverId":   uuid.New(),
					"steamId":    uuid.NewString(),
					"cancelUrl":  "https://rustpm.com/checkout/cancel",
					"successUrl": "https://rustpm.com/checkout/success",
					"priceId":    "price_1LyigBCEcXRU8XL2L6eMGz6Y",
				}
			},
			checkoutEnabled: false,
			firstServerByID: func(t *testing.T, _ *shared) func(context.Context, uuid.UUID) (*model.Server, error) {
				return func(_ context.Context, _ uuid.UUID) (*model.Server, error) {
					require.FailNow(t, "store.FirstServerByID should not be called.")
					return nil, nil
				}
			},
			isServerVipBySteamID: func(t *testing.T, _ *shared) func(context.Context, uuid.UUID, string) (bool, error) {
				return func(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
					require.FailNow(t, "store.IsServerVipBySteamID should not be called.")
					return false, nil
				}
			},
			stageCheckout: func(t *testing.T, _ *shared) func(context.Context, interface{}, time.Time) (string, error) {
				return func(_ context.Context, _ interface{}, _ time.Time) (string, error) {
					require.FailNow(t, "staging.StageCheckout should not be called.")
					return "", nil
				}
			},
			checkoutSession: func(t *testing.T, shared *shared) func(*stripe.CheckoutSessionParams) (string, error) {
				return func(params *stripe.CheckoutSessionParams) (string, error) {
					require.FailNow(t, "stripe.CheckoutSession should not be called.")
					return "", nil
				}
			},
			exp: expected{
				status: http.StatusNotFound,
			},
		},
		"happy path": {
			body: func(shared *shared) map[string]interface{} {
				shared.serverID = uuid.New()
				shared.steamID = uuid.NewString()
				return map[string]interface{}{
					"serverId":   shared.serverID.String(),
					"steamId":    shared.steamID,
					"cancelUrl":  "https://rustpm.com/checkout/cancel",
					"successUrl": "https://rustpm.com/checkout/success",
					"priceId":    "price_1LyigBCEcXRU8XL2L6eMGz6Y",
				}
			},
			checkoutEnabled: true,
			firstServerByID: func(t *testing.T, shared *shared) func(context.Context, uuid.UUID) (*model.Server, error) {
				return func(_ context.Context, id uuid.UUID) (*model.Server, error) {
					require.Equal(t, shared.serverID, id)
					// Server instance is unused, so its attributes do not need to mocked.
					return &model.Server{}, nil
				}
			},
			isServerVipBySteamID: func(t *testing.T, shared *shared) func(context.Context, uuid.UUID, string) (bool, error) {
				return func(_ context.Context, serverID uuid.UUID, steamID string) (bool, error) {
					require.Equal(t, shared.serverID, serverID)
					require.Equal(t, shared.steamID, steamID)

					return false, nil
				}
			},
			stageCheckout: func(t *testing.T, shared *shared) func(context.Context, interface{}, time.Time) (string, error) {
				return func(_ context.Context, checkoutI interface{}, expiresAt time.Time) (string, error) {
					checkout, ok := checkoutI.(*staging.Checkout)
					require.True(t, ok, "checkout is not type *staging.Checkout")

					require.Equal(t, shared.serverID, checkout.ServerID)
					require.Equal(t, shared.steamID, checkout.SteamID)

					shared.expiresAt = time.Now().Add(time.Hour)
					require.WithinDuration(t, shared.expiresAt, expiresAt, time.Second)

					shared.clientReferenceID = uuid.NewString()
					return shared.clientReferenceID, nil
				}
			},
			checkoutSession: func(t *testing.T, shared *shared) func(*stripe.CheckoutSessionParams) (string, error) {
				return func(params *stripe.CheckoutSessionParams) (string, error) {
					require.Equal(t, "https://rustpm.com/checkout/cancel", *params.CancelURL)
					require.Equal(t, "https://rustpm.com/checkout/success", *params.SuccessURL)
					require.Equal(t, stripe.CheckoutSessionModePayment, stripe.CheckoutSessionMode(*params.Mode))
					require.Equal(t, shared.clientReferenceID, *params.ClientReferenceID)
					require.Equal(t, shared.expiresAt.Unix(), *params.ExpiresAt)
					require.Empty(t, *params.Customer)

					require.Len(t, params.LineItems, 1, "expected a single line-item")
					require.Equal(t, istripe.WeeklyVipOneTime, istripe.Price(*params.LineItems[0].Price))
					require.Equal(t, 1, int(*params.LineItems[0].Quantity), "expected a line-item with quantity of 1")
					return "https://stripe.com/checkout", nil
				}
			},
			exp: expected{
				status: http.StatusCreated,
				redirect: Redirect{
					URL: "https://stripe.com/checkout",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			shared := &shared{}

			store := db.NewStoreMock(
				db.WithFirstServerByID(test.firstServerByID(t, shared)),
				db.WithIsServerVipBySteamID(test.isServerVipBySteamID(t, shared)),
			)
			staging := staging.NewClientMock(
				staging.WithStageCheckout(test.stageCheckout(t, shared)),
			)
			stripe := istripe.NewMock(
				istripe.WithCheckoutSession(test.checkoutSession(t, shared)),
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
				staging,
				stream.NewClientMock(),
				stripe,
				sessionMiddleware,
				healthz.NewHTTP(),
				WithCheckout(test.checkoutEnabled),
			)

			buf := new(bytes.Buffer)
			err := json.NewEncoder(buf).Encode(test.body(shared))
			require.Nil(t, err)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/checkout", buf)

			api.Mux.ServeHTTP(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			require.Equal(t, test.exp.status, resp.StatusCode)

			if resp.StatusCode != http.StatusCreated {
				return
			}

			var redirect Redirect
			err = json.NewDecoder(resp.Body).Decode(&redirect)
			require.Nil(t, err)

			require.Equal(t, test.exp.redirect, redirect)

		})
	}
}
