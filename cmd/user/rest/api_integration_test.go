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

	"github.com/tjper/rustcron/cmd/user/admin"
	"github.com/tjper/rustcron/cmd/user/config"
	"github.com/tjper/rustcron/cmd/user/controller"
	"github.com/tjper/rustcron/cmd/user/db"
	"github.com/tjper/rustcron/internal/email"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"

	redisv8 "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	dsn = flag.String(
		"dsn",
		"host=localhost user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=UTC",
		"DSN to be used to connect to user DB.",
	)
	migrations = flag.String(
		"migrations",
		"file://../db/migrations",
		"Migrations to be used to migrate DB to correct schema.",
	)
	redisAddr = flag.String(
		"redis-addr",
		"localhost:6379",
		"Redis address to be used to establish Redis client.",
	)
	redisPassword = flag.String(
		"redis-pass",
		"",
		"Redis password to be used to authenticate with Redis.",
	)
	admins = []string{"rustcron@gmail.com"}
)

func TestIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "rustcron@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		assert.Equal(t, resp.StatusCode, http.StatusCreated)
	})

	// t.Run("create user that already exists", func(t *testing.T) {

	// })

	// t.Run("forgot password", func(t *testing.T) {

	// })

	// t.Run("update password", func(t *testing.T) {

	// })

	// t.Run("update password w/ invalid hash", func(t *testing.T) {

	// })

	// t.Run("fetch session w/ invalid id", func(t *testing.T) {

	// })

	// t.Run("login user w/ invalid credentials", func(t *testing.T) {

	// })

	// t.Run("login user", func(t *testing.T) {

	// })

	// t.Run("update user's password", func(t *testing.T) {

	// })

	// t.Run("resend verification email", func(t *testing.T) {

	// })

	// t.Run("verify email", func(t *testing.T) {

	// })

	// t.Run("verify email w/ invalid hash", func(t *testing.T) {

	// })

	// t.Run("logout user", func(t *testing.T) {

	// })

	// t.Run("logout user w/ invalid session", func(t *testing.T) {

	// })
}

func setup(ctx context.Context, t *testing.T) *suite {
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

	sessionManager := session.NewManager(logger, rdb)

	emailer := email.NewMock()

	ctrl := controller.New(
		sessionManager,
		db.NewStore(logger, dbconn),
		emailer,
		admin.NewAdminSet(admins),
		48*time.Hour,   // 2 days
		7*24*time.Hour, // 1 week
	)

	api := NewAPI(
		logger,
		ctrl,
		ihttp.CookieOptions{
			Domain:   config.CookieDomain(),
			Secure:   config.CookieSecure(),
			SameSite: config.CookieSameSite(),
		},
		sessionManager,
		ihttp.NewSessionMiddleware(
			logger,
			sessionManager,
			48*time.Hour, // 2 days
		),
	)

	return &suite{
		emailer: emailer,
		api:     api.Mux,
	}
}

type suite struct {
	emailer controller.IEmailer
	api     http.Handler
}

// --- helpers ---

func (s suite) request(
	ctx context.Context,
	t *testing.T,
	method string,
	target string,
	body interface{},
) *http.Response {
	buf := new(bytes.Buffer)

	err := json.NewEncoder(buf).Encode(body)
	assert.Nil(t, err)

	req := httptest.NewRequest(method, target, buf)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	s.api.ServeHTTP(rr, req)

	return rr.Result()
}
