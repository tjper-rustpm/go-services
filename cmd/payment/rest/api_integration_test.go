// +build integration

package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/payment/controller"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stripev72 "github.com/stripe/stripe-go/v72"
)

func TestCreateCheckoutSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com")

	t.Run("create subscription checkout session", func(t *testing.T) {
		suite.postSubscriptionCheckoutSession(ctx, t, uuid.New(), uuid.New(), sess)
	})
}

func TestCreateBillingPortalSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com")

	t.Run("create billing portal session", func(t *testing.T) {
		body := map[string]interface{}{
			"returnUrl": "http://rustpm.com",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/billing", body, sess)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusSeeOther, resp.StatusCode)

		session := suite.stripe.PopBillingPortalSession()
		assert.Equal(t, "http://rustpm.com", *session.ReturnURL)
	})
}

func TestCheckoutSessionComplete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com")

	var clientReferenceID uuid.UUID
	t.Run("create subscription checkout session", func(t *testing.T) {
		clientReferenceID = suite.postSubscriptionCheckoutSession(ctx, t, uuid.New(), uuid.New(), sess)
	})

	eventID := uuid.New()
	t.Run("complete checkout session", func(t *testing.T) {
		suite.postCompleteCheckoutSession(ctx, t, eventID, uuid.New(), clientReferenceID, uuid.New(), uuid.New(), sess)

		stagingCheckout, err := suite.staging.FetchCheckout(ctx, clientReferenceID.String())
		assert.Nil(t, err)

		eventI := suite.stream.ReadEvent(ctx, t)
		event, ok := eventI.(*event.SubscriptionCreatedEvent)
		assert.True(t, ok)
		assert.Equal(t, stagingCheckout.ServerID, event.ServerID)
		assert.Equal(t, stagingCheckout.UserID, event.UserID)
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

	sess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com")

	var (
		clientReferenceID uuid.UUID
		serverID          = uuid.New()
	)
	t.Run("create subscription checkout session", func(t *testing.T) {
		clientReferenceID = suite.postSubscriptionCheckoutSession(
			ctx,
			t,
			serverID,
			sess.User.ID,
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
		event, ok := eventI.(*event.SubscriptionCreatedEvent)
		assert.True(t, ok)
		assert.Equal(t, serverID.String(), event.ServerID.String())
		assert.Equal(t, sess.User.ID.String(), event.UserID.String())
		subscriptionID = event.SubscriptionID

		updateFn := func(sess *session.Session) {
			sess.User.Subscriptions = []session.Subscription{
				{ID: event.SubscriptionID, ServerID: serverID},
			}
		}
		_, err := suite.sessions.Manager.UpdateSession(ctx, sess.ID, updateFn)
		assert.Nil(t, err)
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
		assert.Len(t, subs, 1)

		sub := subs[0]
		assert.Equal(t, subscriptionID, sub.ID)
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
		event, ok := eventI.(*event.SubscriptionDeleteEvent)
		assert.True(t, ok)
		assert.Equal(t, subscriptionID, event.SubscriptionID)

		updateFn := func(sess *session.Session) { sess.User.Subscriptions = nil }
		_, err := suite.sessions.Manager.UpdateSession(ctx, sess.ID, updateFn)
		assert.Nil(t, err)
	})

	t.Run("check session has no subscription", func(t *testing.T) {
		subs := suite.getSubscriptions(ctx, t, sess)
		assert.Len(t, subs, 0)
	})
}

func (s suite) postSubscriptionCheckoutSession(
	ctx context.Context,
	t *testing.T,
	serverID uuid.UUID,
	userID uuid.UUID,
	sess *session.Session,
) uuid.UUID {
	t.Helper()

	body := map[string]interface{}{
		"serverId":   serverID,
		"userId":     userID,
		"cancelUrl":  "http://rustpm.com/payment/cancel",
		"successUrl": "http://rustpm.com/payment/success",
		"priceId":    "prod_L1MFlCUj2bk2j0",
	}

	resp := s.Request(ctx, t, s.api, http.MethodPost, "/v1/checkout", body, sess)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)

	stripeCheckout := s.stripe.PopCheckoutSession()
	assert.Equal(t, "http://rustpm.com/payment/cancel", *stripeCheckout.CancelURL)
	assert.Equal(t, "http://rustpm.com/payment/success", *stripeCheckout.SuccessURL)
	assert.Equal(t, string(stripev72.CheckoutSessionModeSubscription), *stripeCheckout.Mode)
	assert.Equal(t, "prod_L1MFlCUj2bk2j0", *stripeCheckout.LineItems[0].Price)
	assert.Equal(t, int64(1), *stripeCheckout.LineItems[0].Quantity)

	stagingCheckout, err := s.staging.FetchCheckout(ctx, *stripeCheckout.ClientReferenceID)
	assert.Nil(t, err)
	assert.Equal(t, stagingCheckout.ServerID, serverID)
	assert.Equal(t, stagingCheckout.UserID, userID)

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

	assert.Equal(t, http.StatusOK, resp.StatusCode)
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

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func (s suite) getSubscriptions(
	ctx context.Context,
	t *testing.T,
	sess *session.Session,
) []Subscription {
	t.Helper()

	resp := s.Request(ctx, t, s.api, http.MethodGet, "/v1/subscriptions", nil, sess)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var subs []Subscription
	err := json.NewDecoder(resp.Body).Decode(&subs)
	assert.Nil(t, err)

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
	require.Nil(t, err)

	staging := staging.NewClient(s.Redis)
	stripe := stripe.NewMock()
	ctrl := controller.New(s.Logger, dbconn, staging, stripe, s.Stream)

	api := NewAPI(
		s.Logger,
		ctrl,
		ihttp.NewSessionMiddleware(s.Logger, sessions.Manager),
		stripe,
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
