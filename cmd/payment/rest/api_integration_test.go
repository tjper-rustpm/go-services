//go:build integration
// +build integration

package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/payment/db"
	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/internal/event"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/integration"
	"github.com/tjper/rustcron/internal/redis"
	"github.com/tjper/rustcron/internal/session"
	"github.com/tjper/rustcron/internal/stream"
	"github.com/tjper/rustcron/internal/stripe"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	stripev72 "github.com/stripe/stripe-go/v72"
)

func TestCreateCheckoutSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com", session.RoleStandard, "steam-id")

	t.Run("create subscription checkout session", func(t *testing.T) {
		suite.postSubscriptionCheckoutSession(ctx, t, uuid.New(), sess)
	})
}

func TestCreateBillingPortalSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com", session.RoleStandard, "steam-id")

	t.Run("create billing portal session", func(t *testing.T) {
		body := map[string]interface{}{
			"returnUrl": "http://rustpm.com",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/billing", body, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusSeeOther, resp.StatusCode)

		session := suite.stripe.PopBillingPortalSession()
		require.Equal(t, "http://rustpm.com", *session.ReturnURL)
	})
}

func TestCheckoutSessionComplete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com", session.RoleStandard, "steam-id")

	var clientReferenceID uuid.UUID
	t.Run("create subscription checkout session", func(t *testing.T) {
		clientReferenceID = suite.postSubscriptionCheckoutSession(ctx, t, uuid.New(), sess)
	})

	eventID := uuid.New()
	t.Run("complete checkout session", func(t *testing.T) {
		suite.postCompleteCheckoutSession(ctx, t, eventID, uuid.New(), clientReferenceID, uuid.New(), uuid.New(), sess)

		stagingCheckout, err := suite.staging.FetchCheckout(ctx, clientReferenceID.String())
		require.Nil(t, err)

		eventI := suite.stream.ReadEvent(ctx, t)
		event, ok := eventI.(*event.InvoicePaidEvent)
		require.True(t, ok)
		require.NotEmpty(t, event.SubscriptionID)
		require.Equal(t, stagingCheckout.ServerID, event.ServerID)
		require.Equal(t, stagingCheckout.SteamID, event.SteamID)
	})

	t.Run("duplicate complete checkout session", func(t *testing.T) {
		suite.postCompleteCheckoutSession(ctx, t, eventID, uuid.New(), clientReferenceID, uuid.New(), uuid.New(), sess)
		suite.stream.AssertNoEvent(ctx, t)
	})

	t.Run("complete checkout session w/ invalid client reference ID", func(t *testing.T) {
		suite.postCompleteCheckoutSession(ctx, t, uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), sess)
		suite.stream.AssertNoEvent(ctx, t)
	})
}

func TestInvoice(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com", session.RoleStandard, "steam-id")

	var (
		clientReferenceID uuid.UUID
		serverID          = uuid.New()
	)
	t.Run("create subscription checkout session", func(t *testing.T) {
		clientReferenceID = suite.postSubscriptionCheckoutSession(
			ctx,
			t,
			serverID,
			sess,
		)
	})

	var (
		subscriptionID       uuid.UUID
		stripeSubscriptionID = uuid.New()
	)
	t.Run("complete checkout session", func(t *testing.T) {
		suite.postCompleteCheckoutSession(
			ctx,
			t,
			uuid.New(),
			uuid.New(),
			clientReferenceID,
			uuid.New(),
			stripeSubscriptionID,
			sess,
		)

		eventI := suite.stream.ReadEvent(ctx, t)
		event, ok := eventI.(*event.InvoicePaidEvent)
		require.True(t, ok)
		require.Equal(t, serverID, event.ServerID)
		require.Equal(t, sess.User.SteamID, event.SteamID)
		subscriptionID = event.SubscriptionID
	})

	t.Run("invoice paid", func(t *testing.T) {
		suite.postInvoice(
			ctx,
			t,
			uuid.New(),
			"invoice.paid",
			uuid.New(),
			"paid",
			stripeSubscriptionID,
			sess,
		)
	})

	t.Run("get session's subscription", func(t *testing.T) {
		subs := suite.getSubscriptions(ctx, t, sess)
		require.Len(t, subs, 1)

		sub := subs[0]
		require.Equal(t, subscriptionID, sub.ID)
	})

	t.Run("invoice payment failed", func(t *testing.T) {
		suite.postInvoice(
			ctx,
			t,
			uuid.New(),
			"invoice.payment_failed",
			uuid.New(),
			"payment_failed",
			stripeSubscriptionID,
			sess,
		)

		eventI := suite.stream.ReadEvent(ctx, t)
		event, ok := eventI.(*event.InvoicePaymentFailureEvent)
		require.True(t, ok)
		require.Equal(t, subscriptionID, event.SubscriptionID)
	})

	t.Run("check session has no subscription", func(t *testing.T) {
		subs := suite.getSubscriptions(ctx, t, sess)
		require.Len(t, subs, 0)
	})
}

func (s suite) postSubscriptionCheckoutSession(
	ctx context.Context,
	t *testing.T,
	serverID uuid.UUID,
	sess *session.Session,
) uuid.UUID {
	t.Helper()

	body := map[string]interface{}{
		"serverId":   serverID,
		"cancelUrl":  "http://rustpm.com/payment/cancel",
		"successUrl": "http://rustpm.com/payment/success",
		"priceId":    "price_1KLJWjCEcXRU8XL2TVKcLGUO",
	}

	resp := s.Request(ctx, t, s.api, http.MethodPost, "/v1/checkout", body, sess)
	defer resp.Body.Close()

	require.Equal(t, http.StatusSeeOther, resp.StatusCode)

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
	require.Equal(t, stagingCheckout.SteamID, sess.User.SteamID)

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
	sess *session.Session,
) {
	t.Helper()

	body := invoiceBody(id, eventType, invoiceID, status, stripeSubscriptionID)

	resp := s.Request(ctx, t, s.api, http.MethodPost, "/v1/stripe", body, sess)
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

func setup(ctx context.Context, t *testing.T) *suite {
	t.Helper()

	redis := redis.InitSuite(ctx, t)
	err := redis.Redis.FlushAll(ctx).Err()
	require.Nil(t, err)

	s := integration.InitSuite(ctx, t)
	sessions := session.InitSuite(ctx, t)
	stream := stream.InitSuite(ctx, t)

	const (
		dsn        = "host=db user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=UTC"
		migrations = "file://../db/migrations"
	)

	dbconn, err := db.Open(dsn)
	require.Nil(t, err)

	err = db.Migrate(dbconn, migrations)
  require.Nil(t, err, "Migrate: %s", err)

	staging := staging.NewClient(s.Redis)
	stripe := stripe.NewMock()

	api := NewAPI(
		s.Logger,
		db.NewStore(dbconn),
		staging,
		s.Stream,
		stripe,
		ihttp.NewSessionMiddleware(s.Logger, sessions.Manager),
	)

	return &suite{
		Suite:    *s,
		sessions: sessions,
		stream:   stream,
		api:      api.Mux,
		staging:  staging,
		stripe:   stripe,
	}
}

type suite struct {
	integration.Suite
	sessions *session.Suite
	stream   *stream.Suite

	api     http.Handler
	staging *staging.Client
	stripe  *stripe.Mock
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
