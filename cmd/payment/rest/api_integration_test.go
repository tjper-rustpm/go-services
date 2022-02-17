// +build integration

package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/payment/controller"
	"github.com/tjper/rustcron/cmd/payment/db"
	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/internal/event"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/rand"
	"github.com/tjper/rustcron/internal/session"
	"github.com/tjper/rustcron/internal/stream"
	"github.com/tjper/rustcron/internal/stripe"

	redisv8 "github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stripev72 "github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
)

var (
	dsn = flag.String(
		"dsn",
		"host=db user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=UTC",
		"DSN to be used to connect to user DB.",
	)
	migrations = flag.String(
		"migrations",
		"file://../db/migrations",
		"Migrations to be used to migrate DB to correct schema.",
	)
	redisAddr = flag.String(
		"redis-addr",
		"redis:6379",
		"Redis address to be used to establish Redis client.",
	)
	redisPassword = flag.String(
		"redis-pass",
		"",
		"Redis password to be used to authenticate with Redis.",
	)
)

func TestCreateCheckoutSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.session(
		ctx,
		t,
		session.User{
			ID:    uuid.New(),
			Email: "rustcron@gmail.com",
			Role:  session.RoleStandard,
		},
	)

	t.Run("create subscription checkout session", func(t *testing.T) {
		suite.postSubscriptionCheckoutSession(
			ctx,
			t,
			uuid.New(),
			uuid.New(),
			sess,
		)
	})
}

func TestCreateBillingPortalSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.session(
		ctx,
		t,
		session.User{
			ID:    uuid.New(),
			Email: "rustcron@gmail.com",
			Role:  session.RoleStandard,
		},
	)

	t.Run("create billing portal session", func(t *testing.T) {
		body := map[string]interface{}{
			"returnUrl": "http://rustpm.com",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/billing", body, cookie(sess))
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

	sess := suite.session(
		ctx,
		t,
		session.User{
			ID:    uuid.New(),
			Email: "rustcron@gmail.com",
			Role:  session.RoleStandard,
		},
	)

	var clientReferenceID uuid.UUID
	t.Run("create subscription checkout session", func(t *testing.T) {
		clientReferenceID = suite.postSubscriptionCheckoutSession(ctx, t, uuid.New(), uuid.New(), sess)
	})

	eventID := uuid.New()
	t.Run("complete checkout session", func(t *testing.T) {
		suite.postCompleteCheckoutSession(
			ctx,
			t,
			eventID,
			uuid.New(),
			clientReferenceID,
			uuid.New(),
			uuid.New(),
			sess,
		)
	})

	t.Run("duplicate complete checkout session", func(t *testing.T) {
		suite.postCompleteCheckoutSession(
			ctx,
			t,
			eventID,
			uuid.New(),
			clientReferenceID,
			uuid.New(),
			uuid.New(),
			sess,
		)
	})

	t.Run("complete checkout session w/ invalid client reference ID", func(t *testing.T) {
		suite.postCompleteCheckoutSession(
			ctx,
			t,
			uuid.New(),
			uuid.New(),
			uuid.New(),
			uuid.New(),
			uuid.New(),
			sess,
		)
	})
}

func TestInvoice(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.session(
		ctx,
		t,
		session.User{
			ID:    uuid.New(),
			Email: "rustcron@gmail.com",
			Role:  session.RoleStandard,
		},
	)

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

		eventI := suite.readEvent(ctx, t)
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
		_, err := suite.sessions.UpdateSession(ctx, sess.ID, updateFn)
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

		eventI := suite.readEvent(ctx, t)
		event, ok := eventI.(*event.SubscriptionDeleteEvent)
		assert.True(t, ok)
		assert.Equal(t, subscriptionID, event.SubscriptionID)

		updateFn := func(sess *session.Session) { sess.User.Subscriptions = nil }
		_, err := suite.sessions.UpdateSession(ctx, sess.ID, updateFn)
		assert.Nil(t, err)
	})

	t.Run("check session has no subscription", func(t *testing.T) {
		subs := suite.getSubscriptions(ctx, t, sess)
		assert.Len(t, subs, 0)
	})
}

func (s suite) readEvent(ctx context.Context, t *testing.T) interface{} {
	t.Helper()

	m, err := s.stream.Read(ctx)
	assert.Nil(t, err)

	eventI, err := event.Parse(m.Payload)
	assert.Nil(t, err)

	err = m.Ack(ctx)
	assert.Nil(t, err)

	return eventI
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

	resp := s.request(ctx, t, http.MethodPost, "/v1/checkout", body, cookie(sess))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)

	checkout := s.stripe.PopCheckoutSession()
	assert.Equal(t, "http://rustpm.com/payment/cancel", *checkout.CancelURL)
	assert.Equal(t, "http://rustpm.com/payment/success", *checkout.SuccessURL)
	assert.Equal(t, string(stripev72.CheckoutSessionModeSubscription), *checkout.Mode)
	assert.Equal(t, "prod_L1MFlCUj2bk2j0", *checkout.LineItems[0].Price)
	assert.Equal(t, int64(1), *checkout.LineItems[0].Quantity)

	return uuid.MustParse(*checkout.ClientReferenceID)
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

	resp := s.request(ctx, t, http.MethodPost, "/v1/stripe", body, cookie(sess))
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

	resp := s.request(ctx, t, http.MethodPost, "/v1/stripe", body, cookie(sess))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func (s suite) getSubscriptions(
	ctx context.Context,
	t *testing.T,
	sess *session.Session,
) []Subscription {
	t.Helper()

	resp := s.request(ctx, t, http.MethodGet, "/v1/subscriptions", nil, cookie(sess))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var subs []Subscription
	err := json.NewDecoder(resp.Body).Decode(&subs)
	assert.Nil(t, err)

	return subs
}

func setup(ctx context.Context, t *testing.T) *suite {
	t.Helper()

	logger := zap.NewNop()

	dbconn, err := db.Open(*dsn)
	require.Nil(t, err)

	err = db.Migrate(dbconn, *migrations)
	require.Nil(t, err)

	rdb := redisv8.NewClient(&redisv8.Options{
		Addr:     *redisAddr,
		Password: *redisPassword,
	})
	err = rdb.Ping(ctx).Err()
	require.Nil(t, err)

	err = rdb.FlushDB(ctx).Err()
	require.Nil(t, err)

	sessionManager := session.NewMock(time.Hour)

	stripe := stripe.NewMock()

	stream, err := stream.Init(ctx, rdb, "test")
	require.Nil(t, err)

	ctrl := controller.New(
		logger,
		dbconn,
		staging.NewClient(rdb),
		stripe,
		stream,
	)

	api := NewAPI(
		logger,
		ctrl,
		ihttp.NewSessionMiddleware(
			logger,
			sessionManager,
		),
		stripe,
	)

	return &suite{
		api:      api.Mux,
		stripe:   stripe,
		stream:   stream,
		sessions: sessionManager,
	}
}

type suite struct {
	api      http.Handler
	stripe   *stripe.Mock
	stream   *stream.Client
	sessions *session.Mock
}

func (s *suite) request(
	ctx context.Context,
	t *testing.T,
	method string,
	target string,
	body interface{},
	cookies ...*http.Cookie,
) *http.Response {
	t.Helper()

	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, target, nil)
	} else {
		buf := new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		assert.Nil(t, err)

		req = httptest.NewRequest(method, target, buf)
	}

	req = req.WithContext(ctx)

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	rr := httptest.NewRecorder()
	s.api.ServeHTTP(rr, req)

	return rr.Result()
}

func (s *suite) session(
	ctx context.Context,
	t *testing.T,
	user session.User,
) *session.Session {
	t.Helper()

	id, err := rand.GenerateString(16)
	require.Nil(t, err)

	sess := session.New(id, user, time.Minute)

	err = s.sessions.CreateSession(ctx, *sess)
	require.Nil(t, err)

	return sess
}

// --- helpers ---

func cookie(sess *session.Session) *http.Cookie {
	return ihttp.Cookie(sess.ID, ihttp.CookieOptions{})
}

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
