package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/payment/controller"
	"github.com/tjper/rustcron/cmd/payment/db"
	"github.com/tjper/rustcron/cmd/payment/staging"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/rand"
	"github.com/tjper/rustcron/internal/session"
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

	subscriptionID := uuid.New()
	t.Run("complete checkout session", func(t *testing.T) {
		suite.postCompleteCheckoutSession(
			ctx,
			t,
			uuid.New(),
			uuid.New(),
			clientReferenceID,
			uuid.New(),
			subscriptionID,
			sess,
		)
	})

	t.Run("invoice paid", func(t *testing.T) {
		suite.postInvoice(
			ctx,
			t,
			uuid.New(),
			"invoice.paid",
			uuid.New(),
			"paid",
			subscriptionID,
			sess,
		)
	})

	t.Run("get paid subscription", func(t *testing.T) {
		subs := suite.getSubscriptions(ctx, t, sess)
		assert.Len(t, subs, 1)

		sub := subs[0]
		assert.Equal(t, sub.ServerID, serverID)
		assert.Equal(t, sub.UserID, sess.User.ID)
		assert.True(t, sub.Active)
	})

	t.Run("invoice payment failed", func(t *testing.T) {
		suite.postInvoice(
			ctx,
			t,
			uuid.New(),
			"invoice.payment_failed",
			uuid.New(),
			"payment_failed",
			subscriptionID,
			sess,
		)
	})

	t.Run("get failed subscription", func(t *testing.T) {
		subs := suite.getSubscriptions(ctx, t, sess)
		assert.Len(t, subs, 1)

		sub := subs[0]
		assert.Equal(t, sub.ServerID, serverID)
		assert.Equal(t, sub.UserID, sess.User.ID)
		assert.False(t, sub.Active)
	})
}

func TestSingleUserParallelSubscriptionsCreation(t *testing.T) {
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

	servers := make([]uuid.UUID, 0)
	testFn := func(t *testing.T) {
		t.Helper()

		var (
			serverID       = uuid.New()
			subscriptionID = uuid.New()
		)
		clientReferenceID := suite.postSubscriptionCheckoutSession(ctx, t, serverID, sess.User.ID, sess)

		suite.postCompleteCheckoutSession(ctx, t, uuid.New(), uuid.New(), clientReferenceID, uuid.New(), subscriptionID, sess)

		suite.postInvoice(ctx, t, uuid.New(), "invoice.paid", uuid.New(), "paid", subscriptionID, sess)

		servers = append(servers, serverID)
	}

	t.Run("create subscription checkout session", func(t *testing.T) {
		for _, processes := range []int{2, 4, 10, 100} {
			t.Run(fmt.Sprintf("%d parallel subscriptions", processes), func(t *testing.T) {
				var wg sync.WaitGroup
				defer wg.Wait()

				for i := 0; i < processes; i++ {
					wg.Add(1)
					go func() {
						testFn(t)
						wg.Done()
					}()
				}
			})
		}
	})

	t.Run("fetch user subscriptions", func(t *testing.T) {
		subs := suite.getSubscriptions(ctx, t, sess)
		assert.Equal(t, len(subs), len(servers))

		sort.Slice(subs, func(i, j int) bool { return subs[i].ServerID.String() < subs[j].ServerID.String() })
		sort.Slice(servers, func(i, j int) bool { return servers[i].String() < servers[j].String() })

		for i := range subs {
			assert.Equal(t, servers[i], subs[i].ServerID)
		}
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
	subscriptionID uuid.UUID,
	sess *session.Session,
) {
	t.Helper()

	body := checkoutSessionCompleteBody(id, checkoutID, clientReferenceID, customerID, subscriptionID)

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
	subscriptionID uuid.UUID,
	sess *session.Session,
) {
	t.Helper()

	body := invoiceBody(id, eventType, invoiceID, status, subscriptionID)

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

	sessionManager := session.NewMock()

	stripe := stripe.NewMock()

	ctrl := controller.New(
		logger,
		dbconn,
		staging.NewClient(rdb),
		stripe,
	)

	api := NewAPI(
		logger,
		ctrl,
		ihttp.NewSessionMiddleware(
			logger,
			sessionManager,
			48*time.Hour, // 2 days
		),
		stripe,
	)

	return &suite{
		api:      api.Mux,
		stripe:   stripe,
		sessions: sessionManager,
	}
}

type suite struct {
	api      http.Handler
	stripe   *stripe.Mock
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

	err = s.sessions.CreateSession(ctx, *sess, time.Minute)
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
	subscriptionID uuid.UUID,
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
					"id": subscriptionID.String(),
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
	subscriptionID uuid.UUID,
) map[string]interface{} {
	return map[string]interface{}{
		"id":   id.String(),
		"type": eventType,
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":     invoiceID.String(),
				"status": status,
				"subscription": map[string]interface{}{
					"id": subscriptionID.String(),
				},
			},
		},
	}
}
