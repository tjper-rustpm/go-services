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

	t.Run("create subscription checkout session", func(t *testing.T) {
		body := map[string]interface{}{
			"cancelUrl":  "http://rustpm.com/payment/cancel",
			"successUrl": "http://rustpm.com/payment/success",
			"priceId":    "prod_L1MFlCUj2bk2j0",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/payment/checkout", body, suite.cookie)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusSeeOther, resp.StatusCode)

		checkout := suite.stripe.PopCheckoutSession()
		assert.Equal(t, "http://rustpm.com/payment/cancel", *checkout.CancelURL)
		assert.Equal(t, "http://rustpm.com/payment/success", *checkout.SuccessURL)
		assert.Equal(t, string(stripev72.CheckoutSessionModeSubscription), *checkout.Mode)
		assert.Equal(t, "prod_L1MFlCUj2bk2j0", *checkout.LineItems[0].Price)
		assert.Equal(t, int64(1), *checkout.LineItems[0].Quantity)
	})
}

func TestCreateBillingPortalSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	t.Run("create billing portal session", func(t *testing.T) {
		body := map[string]interface{}{
			"returnUrl": "http://rustpm.com",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/payment/billing", body, suite.cookie)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusSeeOther, resp.StatusCode)

		session := suite.stripe.PopBillingPortalSession()
		assert.Equal(t, "http://rustpm.com", *session.ReturnURL)
	})
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
	cookie := cookie(ctx, t, sessionManager)

	stripe := stripe.NewMock()

	ctrl := controller.New(
		logger,
		db.NewStore(dbconn),
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
		"",
	)

	return &suite{
		api:      api.Mux,
		stripe:   stripe,
		sessions: sessionManager,
		cookie:   cookie,
	}
}

type suite struct {
	api      http.Handler
	stripe   *stripe.Mock
	sessions *session.Mock
	cookie   *http.Cookie
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

	buf := new(bytes.Buffer)

	err := json.NewEncoder(buf).Encode(body)
	assert.Nil(t, err)

	req := httptest.NewRequest(method, target, buf)
	req = req.WithContext(ctx)

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	rr := httptest.NewRecorder()
	s.api.ServeHTTP(rr, req)

	return rr.Result()
}

// --- helpers ---

func cookie(ctx context.Context, t *testing.T, sm *session.Mock) *http.Cookie {
	id, err := rand.GenerateString(16)
	require.Nil(t, err)

	user := session.User{
		ID:    uuid.New(),
		Email: "rustcron@gmail.com",
		Role:  session.RoleStandard,
	}

	sess := session.New(id, user, time.Minute)

	err = sm.CreateSession(ctx, *sess, time.Minute)
	require.Nil(t, err)

	return ihttp.Cookie(id, ihttp.CookieOptions{})
}
