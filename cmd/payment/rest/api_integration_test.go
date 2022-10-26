//go:build integration
// +build integration

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/payment/db"
	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/cmd/payment/stream"
	"github.com/tjper/rustcron/internal/event"
	"github.com/tjper/rustcron/internal/healthz"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/integration"
	"github.com/tjper/rustcron/internal/rand"
	"github.com/tjper/rustcron/internal/redis"
	"github.com/tjper/rustcron/internal/session"
	istream "github.com/tjper/rustcron/internal/stream"
	"github.com/tjper/rustcron/internal/stripe"
	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	stripev72 "github.com/stripe/stripe-go/v72"
)

func TestIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var (
		alpha   = uuid.New()
		bravo   = uuid.New()
		charlie = uuid.New()
		delta   = uuid.New()
	)

	suite := setup(
		ctx,
		t,
		map[uuid.UUID]*server{
			alpha:   &server{id: alpha, subscriptionLimit: 10, createSubscriptionsCnt: 0, removeSubscriptionsCnt: 0},
			bravo:   &server{id: bravo, subscriptionLimit: 10, createSubscriptionsCnt: 10, removeSubscriptionsCnt: 1},
			charlie: &server{id: charlie, subscriptionLimit: 20, createSubscriptionsCnt: 15, removeSubscriptionsCnt: 10},
			delta:   &server{id: delta, subscriptionLimit: 30, createSubscriptionsCnt: 30, removeSubscriptionsCnt: 30},
		},
	)

	admin := suite.sessions.CreateSession(ctx, t, "rustcron-admin@gmail.com", session.RoleAdmin)
	standard := suite.sessions.CreateSession(ctx, t, "rustcron-standard@gmail.com", session.RoleStandard)

	t.Run("create servers w/ admin user", func(t *testing.T) {
		for id, server := range suite.servers {
			suite.postServer(ctx, t, id, server.subscriptionLimit, admin)
		}
	})

	t.Run("create servers w/ standard user", func(t *testing.T) {
		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/server", nil, standard)
		defer resp.Body.Close()

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("get servers w/ zero active subscriptions", func(t *testing.T) {
		actual := suite.getServers(ctx, t)
		for _, server := range actual {
			expected := suite.servers[server.ID]
			require.Equal(t, expected.subscriptionLimit, server.SubscriptionLimit)
			require.Empty(t, server.ActiveSubscriptions)
		}
	})

	t.Run("update alpha server subscription limit", func(t *testing.T) {
		subscriptionLimit := 100
		body := map[string]interface{}{
			"id": alpha,
			"changes": map[string]interface{}{
				"subscriptionLimit": subscriptionLimit,
			},
		}
		resp := suite.Request(ctx, t, suite.api, http.MethodPatch, "/v1/server", body, admin)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var server model.Server
		err := json.NewDecoder(resp.Body).Decode(&server)
		require.Nil(t, err)

		require.Equal(t, uint16(subscriptionLimit), server.SubscriptionLimit)
	})

	t.Run("reset alpha server subscription limit", func(t *testing.T) {
		subscriptionLimit := 10
		body := map[string]interface{}{
			"id": alpha,
			"changes": map[string]interface{}{
				"subscriptionLimit": subscriptionLimit,
			},
		}
		resp := suite.Request(ctx, t, suite.api, http.MethodPatch, "/v1/server", body, admin)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var server model.Server
		err := json.NewDecoder(resp.Body).Decode(&server)
		require.Nil(t, err)

		require.Equal(t, uint16(subscriptionLimit), server.SubscriptionLimit)
	})

	t.Run("create alpha subscription", func(t *testing.T) {
		suite.testCheckoutSubscribePaidInvoice(ctx, t, alpha)
	})

	t.Run("remove alpha subscription", func(t *testing.T) {
		suite.testBillingPaymentFailureInvoice(ctx, t, alpha)
	})

	t.Run("create subscriptions w/ all servers", func(t *testing.T) {
		for id, server := range suite.servers {
			for i := 0; i < int(server.createSubscriptionsCnt); i++ {
				suite.testCreatePaidSubscription(ctx, t, id)
			}
		}
	})

	t.Run("get servers w/ active subscriptions", func(t *testing.T) {
		actual := suite.getServers(ctx, t)
		for _, server := range actual {
			expected := suite.servers[server.ID]
			require.Equal(t, expected.subscriptionLimit, server.SubscriptionLimit)
			require.Equal(t, expected.createSubscriptionsCnt, server.ActiveSubscriptions)
		}
	})

	t.Run("remove subscriptions w/ all servers", func(t *testing.T) {
		for id, server := range suite.servers {
			for i := 0; i < int(server.removeSubscriptionsCnt); i++ {
				suite.testRemovePaidSubscription(ctx, t, id)
			}
		}
	})

	t.Run("get servers w/ removed subscriptions", func(t *testing.T) {
		actualServers := suite.getServers(ctx, t)
		for _, actualServer := range actualServers {
			expected := suite.servers[actualServer.ID]
			expectedActiveSubscriptions := expected.createSubscriptionsCnt - expected.removeSubscriptionsCnt

			require.Equal(t, expected.subscriptionLimit, actualServer.SubscriptionLimit)
			require.Equal(t, expectedActiveSubscriptions, actualServer.ActiveSubscriptions)
		}
	})
}

func TestDisabledCheckoutIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	alpha := uuid.New()

	suite := setup(
		ctx,
		t,
		map[uuid.UUID]*server{
			alpha: &server{id: alpha, subscriptionLimit: 10, createSubscriptionsCnt: 0, removeSubscriptionsCnt: 0},
		},
		WithCheckout(false),
	)

	admin := suite.sessions.CreateSession(ctx, t, "rustcron-admin@gmail.com", session.RoleAdmin)
	standard := suite.sessions.CreateSession(ctx, t, "rustcron-standard@gmail.com", session.RoleStandard)

	t.Run("create servers w/ admin user", func(t *testing.T) {
		for id, server := range suite.servers {
			suite.postServer(ctx, t, id, server.subscriptionLimit, admin)
		}
	})

	t.Run("create checkout w/ checkouts disabled", func(t *testing.T) {
		steamID, err := rand.GenerateString(16)
		require.Nil(t, err)

		body := map[string]interface{}{
			"serverId":   alpha,
			"steamId":    steamID,
			"cancelUrl":  "http://rustpm.com/payment/cancel",
			"successUrl": "http://rustpm.com/payment/success",
			"priceId":    "price_1KLJWjCEcXRU8XL2TVKcLGUO",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/checkout", body, standard)
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("get servers w/ checkouts disabled", func(t *testing.T) {
		servers := suite.getServers(ctx, t)
		require.Equal(t, 0, len(servers))
	})
}

func (s suite) testCreatePaidSubscription(ctx context.Context, t *testing.T, serverID uuid.UUID) {
	t.Helper()

	steamID, err := rand.GenerateString(16)
	require.Nil(t, err)

	sess := s.sessions.CreateSession(ctx, t, fmt.Sprintf("email-%s@email.com", steamID), session.RoleStandard)
	clientReferenceID := s.postSubscriptionCheckoutSession(ctx, t, serverID, steamID, sess)

	stripeSubscriptionID := uuid.New()
	eventID := uuid.New()
	s.postCompleteCheckoutSession(ctx, t, eventID, uuid.New(), clientReferenceID, uuid.New(), stripeSubscriptionID, sess)
	s.validateStripeWebhookEvent(ctx, t)

	s.postInvoice(ctx, t, uuid.New(), "invoice.paid", uuid.New(), "paid", stripeSubscriptionID)
	s.validateStripeWebhookEvent(ctx, t)

	eventI := s.stream.ReadEvent(ctx, t)
	event, ok := eventI.(*event.InvoicePaidEvent)
	require.True(t, ok)
	require.Equal(t, serverID, event.ServerID)
	require.Equal(t, steamID, event.SteamID)
	subscriptionID := event.SubscriptionID

	s.servers[serverID].subscriptions = append(
		s.servers[serverID].subscriptions,
		subscription{
			id:       subscriptionID,
			stripeID: stripeSubscriptionID,
			sess:     sess,
		},
	)

	subs := s.getSubscriptions(ctx, t, sess)
	require.Len(t, subs, 1)

	sub := subs[0]
	require.Equal(t, subscriptionID, sub.ID)
	require.Equal(t, serverID, sub.ServerID)
	require.Equal(t, model.InvoiceStatusPaid, sub.Status)
}

func (s suite) testRemovePaidSubscription(ctx context.Context, t *testing.T, serverID uuid.UUID) {
	t.Helper()

	require.NotEmpty(t, s.servers[serverID].subscriptions[0])
	subscription := s.servers[serverID].subscriptions[0]
	s.servers[serverID].subscriptions = s.servers[serverID].subscriptions[1:]

	s.postInvoice(ctx, t, uuid.New(), "invoice.payment_failed", uuid.New(), "payment_failed", subscription.stripeID)
	s.validateStripeWebhookEvent(ctx, t)

	eventI := s.stream.ReadEvent(ctx, t)
	event, ok := eventI.(*event.InvoicePaymentFailureEvent)
	require.True(t, ok)
	require.Equal(t, subscription.id, event.SubscriptionID)

	subs := s.getSubscriptions(ctx, t, subscription.sess)
	require.Len(t, subs, 1)

	sub := subs[0]
	require.Equal(t, subscription.id, sub.ID)
	require.Equal(t, serverID, sub.ServerID)
	require.Equal(t, model.InvoiceStatusPaymentFailed, sub.Status)
}

func (s suite) testCheckoutSubscribePaidInvoice(ctx context.Context, t *testing.T, serverID uuid.UUID) {
	t.Helper()

	steamID, err := rand.GenerateString(16)
	require.Nil(t, err)

	t.Run("create subscription checkout session w/ no session", func(t *testing.T) {
		resp := s.Request(ctx, t, s.api, http.MethodPost, "/v1/checkout", nil)
		defer resp.Body.Close()

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	sess := s.sessions.CreateSession(ctx, t, fmt.Sprintf("email-%s@email.com", steamID), session.RoleStandard)

	var clientReferenceID uuid.UUID
	t.Run("create subscription checkout session", func(t *testing.T) {
		clientReferenceID = s.postSubscriptionCheckoutSession(ctx, t, serverID, steamID, sess)
	})

	var (
		subscriptionID       uuid.UUID
		stripeSubscriptionID = uuid.New()
		eventID              = uuid.New()
	)
	t.Run("complete checkout session", func(t *testing.T) {
		s.postCompleteCheckoutSession(ctx, t, eventID, uuid.New(), clientReferenceID, uuid.New(), stripeSubscriptionID, sess)
		s.validateStripeWebhookEvent(ctx, t)
	})

	t.Run("invoice paid", func(t *testing.T) {
		s.postInvoice(ctx, t, uuid.New(), "invoice.paid", uuid.New(), "paid", stripeSubscriptionID)
		s.validateStripeWebhookEvent(ctx, t)

		eventI := s.stream.ReadEvent(ctx, t)
		event, ok := eventI.(*event.InvoicePaidEvent)
		require.True(t, ok)
		require.Equal(t, serverID, event.ServerID)
		require.Equal(t, steamID, event.SteamID)
		subscriptionID = event.SubscriptionID

		s.servers[serverID].subscriptions = append(
			s.servers[serverID].subscriptions,
			subscription{
				id:       subscriptionID,
				stripeID: stripeSubscriptionID,
				sess:     sess,
			},
		)
	})

	t.Run("get session's subscription", func(t *testing.T) {
		subs := s.getSubscriptions(ctx, t, sess)
		require.Len(t, subs, 1)

		sub := subs[0]
		require.Equal(t, subscriptionID, sub.ID)
		require.Equal(t, serverID, sub.ServerID)
		require.Equal(t, model.InvoiceStatusPaid, sub.Status)
	})

	t.Run("create subscription checkout session for server already subscribed to", func(t *testing.T) {
		body := map[string]interface{}{
			"serverId":   serverID,
			"steamId":    steamID,
			"cancelUrl":  "http://rustpm.com/payment/cancel",
			"successUrl": "http://rustpm.com/payment/success",
			"priceId":    "price_1KLJWjCEcXRU8XL2TVKcLGUO",
		}

		resp := s.Request(ctx, t, s.api, http.MethodPost, "/v1/checkout", body, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("duplicate complete checkout session", func(t *testing.T) {
		s.postCompleteCheckoutSession(ctx, t, eventID, uuid.New(), clientReferenceID, uuid.New(), uuid.New(), sess)
		s.validateStripeWebhookEvent(ctx, t)
		s.stream.AssertNoEvent(ctx, t)
	})

	t.Run("complete checkout session w/ invalid client reference ID", func(t *testing.T) {
		s.postCompleteCheckoutSession(ctx, t, uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), sess)
		s.validateStripeWebhookEvent(ctx, t)
		s.stream.AssertNoEvent(ctx, t)
	})
}

func (s suite) testBillingPaymentFailureInvoice(ctx context.Context, t *testing.T, serverID uuid.UUID) {
	t.Helper()

	require.NotEmpty(t, s.servers[serverID].subscriptions[0])
	subscription := s.servers[serverID].subscriptions[0]
	s.servers[serverID].subscriptions = s.servers[serverID].subscriptions[1:]

	t.Run("create billing portal session", func(t *testing.T) {
		body := map[string]interface{}{
			"returnUrl": "http://rustpm.com",
		}

		resp := s.Request(ctx, t, s.api, http.MethodPost, "/v1/billing", body, subscription.sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		session := s.stripe.PopBillingPortalSession()
		require.Equal(t, "http://rustpm.com", *session.ReturnURL)
	})

	t.Run("invoice payment failed", func(t *testing.T) {
		s.postInvoice(ctx, t, uuid.New(), "invoice.payment_failed", uuid.New(), "payment_failed", subscription.stripeID)
		s.validateStripeWebhookEvent(ctx, t)

		eventI := s.stream.ReadEvent(ctx, t)
		event, ok := eventI.(*event.InvoicePaymentFailureEvent)
		require.True(t, ok)
		require.Equal(t, subscription.id, event.SubscriptionID)
	})

	t.Run("check session has no subscription", func(t *testing.T) {
		subs := s.getSubscriptions(ctx, t, subscription.sess)
		require.Len(t, subs, 1)

		sub := subs[0]
		require.Equal(t, subscription.id, sub.ID)
		require.Equal(t, serverID, sub.ServerID)
		require.Equal(t, model.InvoiceStatusPaymentFailed, sub.Status)
	})
}

func (s suite) getServers(
	ctx context.Context,
	t *testing.T,
) model.Servers {
	t.Helper()

	resp := s.Request(ctx, t, s.api, http.MethodGet, "/v1/servers", nil)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	var servers model.Servers
	err := json.NewDecoder(resp.Body).Decode(&servers)
	require.Nil(t, err)

	return servers
}

func (s suite) postServer(
	ctx context.Context,
	t *testing.T,
	serverID uuid.UUID,
	subscriptionLimit uint16,
	sess *session.Session,
) {
	t.Helper()

	body := map[string]interface{}{
		"id":                serverID,
		"subscriptionLimit": subscriptionLimit,
	}

	resp := s.Request(ctx, t, s.api, http.MethodPost, "/v1/server", body, sess)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var server model.Server
	err := json.NewDecoder(resp.Body).Decode(&server)
	require.Nil(t, err)

	require.Equal(t, serverID, server.ID)
	require.Equal(t, subscriptionLimit, server.SubscriptionLimit)
}

func (s suite) postSubscriptionCheckoutSession(
	ctx context.Context,
	t *testing.T,
	serverID uuid.UUID,
	steamID string,
	sess *session.Session,
) uuid.UUID {
	t.Helper()

	body := map[string]interface{}{
		"serverId":   serverID,
		"steamId":    steamID,
		"cancelUrl":  "http://rustpm.com/payment/cancel",
		"successUrl": "http://rustpm.com/payment/success",
		"priceId":    "price_1KLJWjCEcXRU8XL2TVKcLGUO",
	}

	resp := s.Request(ctx, t, s.api, http.MethodPost, "/v1/checkout", body, sess)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	stripeCheckout := s.stripe.PopCheckoutSession()
	require.Equal(t, "http://rustpm.com/payment/cancel", *stripeCheckout.CancelURL)
	require.Equal(t, "http://rustpm.com/payment/success", *stripeCheckout.SuccessURL)
	require.Equal(t, string(stripev72.CheckoutSessionModeSubscription), *stripeCheckout.Mode)
	require.Equal(t, "price_1KLJWjCEcXRU8XL2TVKcLGUO", *stripeCheckout.LineItems[0].Price)
	require.Equal(t, int64(1), *stripeCheckout.LineItems[0].Quantity)

	stagingCheckout, err := s.staging.FetchCheckout(ctx, *stripeCheckout.ClientReferenceID)
	require.Nil(t, err)
	require.Equal(t, stagingCheckout.ServerID, serverID)
	require.Equal(t, stagingCheckout.UserID, sess.User.ID)
	require.Equal(t, stagingCheckout.SteamID, steamID)

	return uuid.MustParse(*stripeCheckout.ClientReferenceID)
}

func (s suite) postCompleteCheckoutSession(
	ctx context.Context,
	t *testing.T,
	id,
	checkoutID,
	clientReferenceID,
	customerID,
	stripeSubscriptionID uuid.UUID,
	sess *session.Session,
) {
	t.Helper()

	body := checkoutSessionCompleteBody(id, checkoutID, clientReferenceID, customerID, stripeSubscriptionID)

	resp := s.Request(ctx, t, s.api, http.MethodPost, "/v1/stripe", body, sess)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func (s suite) postInvoice(
	ctx context.Context,
	t *testing.T,
	id uuid.UUID,
	eventType string,
	invoiceID uuid.UUID,
	status string,
	stripeSubscriptionID uuid.UUID,
) {
	t.Helper()

	body := invoiceBody(id, eventType, invoiceID, status, stripeSubscriptionID)

	resp := s.Request(ctx, t, s.api, http.MethodPost, "/v1/stripe", body)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func (s suite) getSubscriptions(
	ctx context.Context,
	t *testing.T,
	sess *session.Session,
) []Subscription {
	t.Helper()

	resp := s.Request(ctx, t, s.api, http.MethodGet, "/v1/subscriptions", nil, sess)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var subs []Subscription
	err := json.NewDecoder(resp.Body).Decode(&subs)
	require.Nil(t, err)

	return subs
}

func (s suite) validateStripeWebhookEvent(ctx context.Context, t *testing.T) {
	t.Helper()

	eventI := s.stream.ReadEvent(ctx, t)
	_, ok := eventI.(*event.StripeWebhookEvent)
	require.True(t, ok)
}

func setup(ctx context.Context, t *testing.T, servers map[uuid.UUID]*server, options ...Option) *suite {
	t.Helper()
	logger := zap.NewNop()

	redis := redis.InitSuite(ctx, t)
	err := redis.Redis.FlushAll(ctx).Err()
	require.Nil(t, err)

	s := integration.InitSuite(ctx, t)
	sessions := session.InitSuite(ctx, t)
	streamSuite := istream.InitSuite(ctx, t)

	const (
		dsn        = "host=db user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=UTC"
		migrations = "file://../db/migrations"
	)

	dbconn, err := db.Open(dsn)
	require.Nil(t, err)

	err = db.Migrate(dbconn, migrations)
	require.Nil(t, err, "Migrate: %s", err)

	store := db.NewStore(dbconn)
	staging := staging.NewClient(s.Redis)
	stripe := stripe.NewMock()

	streamClient, err := istream.Init(ctx, logger, redis.Redis, "payment")
	require.Nil(t, err)

	streamHandler := stream.NewHandler(
		logger,
		staging,
		store,
		streamClient,
	)
	go func() {
		err := streamHandler.Launch(ctx)
		require.ErrorIs(t, err, context.Canceled)
	}()

	healthz := healthz.NewHTTP()

	api := NewAPI(
		logger,
		store,
		staging,
		streamClient,
		stripe,
		ihttp.NewSessionMiddleware(logger, sessions.Manager),
		healthz,
		options...,
	)

	return &suite{
		Suite:    *s,
		sessions: sessions,
		stream:   streamSuite,
		api:      api.Mux,
		staging:  staging,
		stripe:   stripe,
		servers:  servers,
	}
}

type suite struct {
	integration.Suite
	sessions *session.Suite
	stream   *istream.Suite

	api     http.Handler
	staging *staging.Client
	stripe  *stripe.Mock

	servers map[uuid.UUID]*server
	alpha   *server
}

type server struct {
	id                     uuid.UUID
	subscriptions          []subscription
	subscriptionLimit      uint16
	createSubscriptionsCnt uint16
	removeSubscriptionsCnt uint16
}

type subscription struct {
	id       uuid.UUID
	stripeID uuid.UUID
	sess     *session.Session
}

// --- helpers ---

func checkoutSessionCompleteBody(
	id,
	checkoutID,
	clientReferenceID,
	customerID,
	stripeSubscriptionID uuid.UUID,
) map[string]interface{} {
	return map[string]interface{}{
		"id":   id.String(),
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":                  checkoutID.String(),
				"client_reference_id": clientReferenceID.String(),
				"customer": map[string]interface{}{
					"id": customerID.String(),
				},
				"subscription": map[string]interface{}{
					"id": stripeSubscriptionID.String(),
				},
			},
		},
	}
}

func invoiceBody(
	id uuid.UUID,
	eventType string,
	invoiceID uuid.UUID,
	status string,
	stripeSubscriptionID uuid.UUID,
) map[string]interface{} {
	return map[string]interface{}{
		"id":   id.String(),
		"type": eventType,
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":     invoiceID.String(),
				"status": status,
				"subscription": map[string]interface{}{
					"id": stripeSubscriptionID.String(),
				},
			},
		},
	}
}
