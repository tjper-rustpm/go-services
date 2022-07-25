//go:build integration
// +build integration

package rest

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/user/admins"
	"github.com/tjper/rustcron/cmd/user/config"
	"github.com/tjper/rustcron/cmd/user/controller"
	"github.com/tjper/rustcron/cmd/user/db"
	email "github.com/tjper/rustcron/internal/email"
	"github.com/tjper/rustcron/internal/healthz"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/integration"
	"github.com/tjper/rustcron/internal/session"

	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "create-user@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotEmpty(t, suite.emailer.VerifyEmailHash("create-user@gmail.com"))
	})

	t.Run("create user that already exists", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "create-user@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("create user w/ invalid email", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "rustcron",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create user w/ invalid password", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "create-user@gmail.com",
			"password": "invalid password",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestForgotPassword(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "forgot-password@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotEmpty(t, suite.emailer.VerifyEmailHash("forgot-password@gmail.com"))
	})

	t.Run("forgot password", func(t *testing.T) {
		body := map[string]interface{}{
			"email": "forgot-password@gmail.com",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/forgot-password", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotEmpty(t, suite.emailer.PasswordResetHash("forgot-password@gmail.com"))
	})

	t.Run("forgot password w/ unknown email", func(t *testing.T) {
		body := map[string]interface{}{
			"email": "unknown@email.com",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/forgot-password", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("update password", func(t *testing.T) {
		body := map[string]interface{}{
			"hash":     suite.emailer.PasswordResetHash("forgot-password@gmail.com"),
			"password": "1UpdatedPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/reset-password", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("update password w/ invalid hash", func(t *testing.T) {
		body := map[string]interface{}{
			"hash":     "invalid hash",
			"password": "1UpdatedPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/reset-password", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestLogin(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "login@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotEmpty(t, suite.emailer.VerifyEmailHash("login@gmail.com"))
	})

	t.Run("fetch session w/ invalid id", func(t *testing.T) {
		resp := suite.Request(ctx, t, suite.api, http.MethodGet, "/v1/user/session", nil)
		defer resp.Body.Close()

		require.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("login user w/ invalid credentials", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "invalid@email.com",
			"password": "1InvalidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/login", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	var sess *session.Session
	t.Run("login user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "login@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/login", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		sess = suite.session(ctx, t, resp.Cookies())
	})

	t.Run("fetch session", func(t *testing.T) {
		resp := suite.Request(ctx, t, suite.api, http.MethodGet, "/v1/user/session", nil, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestUpdateUserPassword(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "update-user-password@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotEmpty(t, suite.emailer.VerifyEmailHash("update-user-password@gmail.com"))
	})

	var sess *session.Session
	t.Run("login user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "update-user-password@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/login", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		sess = suite.session(ctx, t, resp.Cookies())
	})

	t.Run("update user's password w/ invalid password", func(t *testing.T) {
		body := map[string]interface{}{
			"currentPassword": "1ValidPassword",
			"newPassword":     "invalid-password",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/update-password", body, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("update user's password", func(t *testing.T) {
		body := map[string]interface{}{
			"currentPassword": "1ValidPassword",
			"newPassword":     "1UpdatedPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/update-password", body, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("fetch session", func(t *testing.T) {
		resp := suite.Request(ctx, t, suite.api, http.MethodGet, "/v1/user/session", nil, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func TestVerifyEmail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "verify-email@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotEmpty(t, suite.emailer.VerifyEmailHash("verify-email@gmail.com"))
	})

	var sess *session.Session
	t.Run("login user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "verify-email@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/login", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		sess = suite.session(ctx, t, resp.Cookies())
	})

	t.Run("resend verification email", func(t *testing.T) {
		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/resend-verification-email", nil, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotEmpty(t, suite.emailer.VerifyEmailHash("verify-email@gmail.com"))
	})

	t.Run("verify email w/ invalid hash", func(t *testing.T) {
		body := map[string]interface{}{
			"hash": "invalidhash",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/verify-email", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("verify email", func(t *testing.T) {
		body := map[string]interface{}{
			"hash": suite.emailer.VerifyEmailHash("verify-email@gmail.com"),
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/verify-email", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("resend verification email w/ already verified email", func(t *testing.T) {
		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/resend-verification-email", nil, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

func TestLogout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "logout@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotEmpty(t, suite.emailer.VerifyEmailHash("logout@gmail.com"))
	})

	var sess *session.Session
	t.Run("login", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "logout@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/login", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		sess = suite.session(ctx, t, resp.Cookies())
	})

	t.Run("fetch session", func(t *testing.T) {
		resp := suite.Request(ctx, t, suite.api, http.MethodGet, "/v1/user/session", nil, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("logout", func(t *testing.T) {
		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/logout", nil, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("fetch session", func(t *testing.T) {
		resp := suite.Request(ctx, t, suite.api, http.MethodGet, "/v1/user/session", nil, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func TestLogoutAll(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	t.Run("create user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "logout-all@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotEmpty(t, suite.emailer.VerifyEmailHash("logout-all@gmail.com"))
	})

	var sess *session.Session
	t.Run("login", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "logout-all@gmail.com",
			"password": "1ValidPassword",
		}

		resp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/login", body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		sess = suite.session(ctx, t, resp.Cookies())
	})

	t.Run("fetch session", func(t *testing.T) {
		resp := suite.Request(ctx, t, suite.api, http.MethodGet, "/v1/user/session", nil, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("login & logout all", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    "logout-all@gmail.com",
			"password": "1ValidPassword",
		}

		loginResp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/login", body)
		defer loginResp.Body.Close()

		require.Equal(t, http.StatusCreated, loginResp.StatusCode)

		logoutAllResp := suite.Request(ctx, t, suite.api, http.MethodPost, "/v1/user/logout-all", body, sess)
		defer logoutAllResp.Body.Close()

		require.Equal(t, http.StatusCreated, logoutAllResp.StatusCode)

		resp := suite.Request(ctx, t, suite.api, http.MethodGet, "/v1/user/session", nil, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("fetch session afters logout all", func(t *testing.T) {
		resp := suite.Request(ctx, t, suite.api, http.MethodGet, "/v1/user/session", nil, sess)
		defer resp.Body.Close()

		require.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func setup(
	ctx context.Context,
	t *testing.T,
) *suite {
	t.Helper()

	s := integration.InitSuite(ctx, t)
	sessions := session.InitSuite(ctx, t)

	const (
		dsn        = "host=db user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=UTC"
		migrations = "file://../db/migrations"
	)

	dbconn, err := db.Open(dsn)
	require.Nil(t, err)

	err = db.Migrate(dbconn, migrations)
	require.Nil(t, err)

	emailer := email.NewMock()

	ctrl := controller.New(
		db.NewStore(s.Logger, dbconn),
		emailer,
		admins.New([]string{"rustcron@gmail.com"}),
	)

	healthz := healthz.NewHTTP()

	api := NewAPI(
		s.Logger,
		ctrl,
		ihttp.CookieOptions{
			Domain:   config.CookieDomain(),
			Secure:   config.CookieSecure(),
			SameSite: config.CookieSameSite(),
		},
		sessions.Manager,
		ihttp.NewSessionMiddleware(s.Logger, sessions.Manager),
		healthz,
	)

	return &suite{
		Suite:    *s,
		sessions: sessions,
		emailer:  emailer,
		api:      api.Mux,
	}
}

type suite struct {
	integration.Suite
	sessions *session.Suite

	emailer *email.Mock
	api     http.Handler
}

func (s suite) session(
	ctx context.Context,
	t *testing.T,
	cookies []*http.Cookie,
) *session.Session {
	t.Helper()

	sessID := cookies[0].Value

	sess, err := s.sessions.Manager.RetrieveSession(ctx, sessID)
	require.Nil(t, err)

	return sess
}
