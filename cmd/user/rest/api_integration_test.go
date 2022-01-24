// +build userintegration

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
	emailpkg "github.com/tjper/rustcron/internal/email"
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
	admins = []string{"rustcron@gmail.com"}
)

func TestCreateUser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(
		ctx,
		t,
		"create-user@gmail.com",
	)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.NotEmpty(t, suite.emailer.VerifyEmailHash(suite.email))
	})

	t.Run("create user that already exists", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("create user w/ invalid email", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "rustcron",
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create user w/ invalid password", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "invalid password",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestForgotPassword(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(
		ctx,
		t,
		"forgot-password@gmail.com",
	)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.NotEmpty(t, suite.emailer.VerifyEmailHash(suite.email))
	})

	t.Run("forgot password", func(t *testing.T) {
		body := map[string]interface{}{
			"email": suite.email,
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/forgot-password", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.NotEmpty(t, suite.emailer.PasswordResetHash(suite.email))
	})

	t.Run("forgot password w/ unknown email", func(t *testing.T) {
		body := map[string]interface{}{
			"email": "unknown@email.com",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/forgot-password", body)

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("update password", func(t *testing.T) {
		body := map[string]interface{}{
			"hash":     suite.emailer.PasswordResetHash(suite.email),
			"password": "1UpdatedPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/reset-password", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("update password w/ invalid hash", func(t *testing.T) {
		body := map[string]interface{}{
			"hash":     "invalid hash",
			"password": "1UpdatedPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/reset-password", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestLogin(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(
		ctx,
		t,
		"login@gmail.com",
	)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.NotEmpty(t, suite.emailer.VerifyEmailHash(suite.email))
	})

	t.Run("fetch session w/ invalid id", func(t *testing.T) {
		resp := suite.request(ctx, t, http.MethodGet, "/v1/user/session", nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("login user w/ invalid credentials", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "invalid@email.com",
			"password": "1InvalidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/login", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	var cookies []*http.Cookie
	t.Run("login user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/login", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		cookies = resp.Cookies()
	})

	t.Run("fetch session", func(t *testing.T) {
		resp := suite.request(ctx, t, http.MethodGet, "/v1/user/session", nil, cookies...)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestUpdateUserPassword(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(
		ctx,
		t,
		"update-user-password@gmail.com",
	)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.NotEmpty(t, suite.emailer.VerifyEmailHash(suite.email))
	})

	var cookies []*http.Cookie
	t.Run("login user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/login", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		cookies = resp.Cookies()
	})

	t.Run("update user's password w/ invalid password", func(t *testing.T) {
		body := map[string]interface{}{
			"currentPassword": "1ValidPassword",
			"newPassword":     "invalid-password",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/update-password", body, cookies...)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("update user's password", func(t *testing.T) {
		body := map[string]interface{}{
			"currentPassword": "1ValidPassword",
			"newPassword":     "1UpdatedPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/update-password", body, cookies...)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("fetch session", func(t *testing.T) {
		resp := suite.request(ctx, t, http.MethodGet, "/v1/user/session", nil, cookies...)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func TestVerifyEmail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(
		ctx,
		t,
		"verify-email@gmail.com",
	)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.NotEmpty(t, suite.emailer.VerifyEmailHash(suite.email))
	})

	var cookies []*http.Cookie
	t.Run("login user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/login", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		cookies = resp.Cookies()
	})

	t.Run("resend verification email", func(t *testing.T) {
		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/resend-verification-email", nil, cookies...)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.NotEmpty(t, suite.emailer.VerifyEmailHash(suite.email))
	})

	t.Run("verify email w/ invalid hash", func(t *testing.T) {
		body := map[string]interface{}{
			"hash": "invalidhash",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/verify-email", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("verify email", func(t *testing.T) {
		body := map[string]interface{}{
			"hash": suite.emailer.VerifyEmailHash(suite.email),
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/verify-email", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("resend verification email w/ already verified email", func(t *testing.T) {
		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/resend-verification-email", nil, cookies...)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

func TestLogout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(
		ctx,
		t,
		"logout@gmail.com",
	)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.NotEmpty(t, suite.emailer.VerifyEmailHash(suite.email))
	})

	var cookies []*http.Cookie
	t.Run("login", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/login", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		cookies = resp.Cookies()
	})

	t.Run("fetch session", func(t *testing.T) {
		resp := suite.request(ctx, t, http.MethodGet, "/v1/user/session", nil, cookies...)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("logout", func(t *testing.T) {
		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/logout", nil, cookies...)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("fetch session", func(t *testing.T) {
		resp := suite.request(ctx, t, http.MethodGet, "/v1/user/session", nil, cookies...)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func TestLogoutAll(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t, "logout-all@gmail.com")

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.NotEmpty(t, suite.emailer.VerifyEmailHash(suite.email))
	})

	var cookies []*http.Cookie
	t.Run("login", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		resp := suite.request(ctx, t, http.MethodPost, "/v1/user/login", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		cookies = resp.Cookies()
	})

	t.Run("fetch session", func(t *testing.T) {
		resp := suite.request(ctx, t, http.MethodGet, "/v1/user/session", nil, cookies...)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("login & logout all", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    suite.email,
			"password": "1ValidPassword",
		}

		loginResp := suite.request(ctx, t, http.MethodPost, "/v1/user/login", body)
		defer loginResp.Body.Close()

		assert.Equal(t, http.StatusCreated, loginResp.StatusCode)

		logoutAllResp := suite.request(ctx, t, http.MethodPost, "/v1/user/logout-all", body, loginResp.Cookies()...)
		defer logoutAllResp.Body.Close()

		assert.Equal(t, http.StatusCreated, logoutAllResp.StatusCode)

		resp := suite.request(ctx, t, http.MethodGet, "/v1/user/session", nil, loginResp.Cookies()...)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("fetch session afters logout all", func(t *testing.T) {
		resp := suite.request(ctx, t, http.MethodGet, "/v1/user/session", nil, cookies...)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func setup(
	ctx context.Context,
	t *testing.T,
	email string,
) *suite {
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

	emailer := emailpkg.NewMock()

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
		email:   email,
		emailer: emailer,
		api:     api.Mux,
	}
}

type suite struct {
	emailer *email.Mock
	api     http.Handler

	email   string
	cookies []*http.Cookie
}

func (s *suite) request(
	ctx context.Context,
	t *testing.T,
	method string,
	target string,
	body interface{},
	cookies ...*http.Cookie,
) *http.Response {
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
