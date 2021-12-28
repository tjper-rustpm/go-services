package rest

import (
	"context"
	"fmt"
	http "net/http"
	"time"

	"github.com/tjper/rustcron/cmd/user/controller"
	"github.com/tjper/rustcron/cmd/user/model"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IController interface {
	CreateUser(context.Context, controller.CreateUserInput) (*model.User, error)
	User(context.Context, uuid.UUID) (*model.User, error)
	UpdateUserPassword(context.Context, controller.UpdateUserPasswordInput) (*model.User, error)

	LoginUser(context.Context, controller.LoginUserInput) (*controller.LoginUserOutput, error)
	LogoutUserSession(context.Context, session.Session) error
	LogoutAllUserSessions(context.Context, fmt.Stringer) error

	VerifyEmail(context.Context, string) (*model.User, error)
	RequestPasswordReset(context.Context, string) error
	ResetPassword(context.Context, string, string) error
	ResendEmailVerification(context.Context, uuid.UUID) (*model.User, error)
}

func NewAPI(
	logger *zap.Logger,
	ctrl IController,
	cookieOptions ihttp.CookieOptions,
	sessionManager ihttp.ISessionManager,
	sessionExpiration time.Duration,
) *API {
	api := API{
		Mux:            chi.NewRouter(),
		logger:         logger,
		ctrl:           ctrl,
		cookieOptions:  cookieOptions,
		sessionManager: sessionManager,
	}

	api.Mux.Use(
		middleware.RequestLogger(ihttp.NewZapLogFormatter(logger)),
	)

	api.Mux.Route("/v1", func(router chi.Router) {
		router.Method(http.MethodGet, "/user/session", Session{API: api})
		router.Method(http.MethodPost, "/user/forgot-password", ForgotPassword{API: api})
		router.Method(http.MethodPost, "/user/verify-email", VerifyEmail{API: api})
		router.Method(http.MethodPost, "/user/reset-password", ResetPassword{API: api})

		router.Group(func(router chi.Router) {
			router.Use(ihttp.EnsureDuration(2 * time.Second))

			router.Method(http.MethodPost, "/user", CreateUser{API: api})
			router.Method(http.MethodPost, "/user/login", LoginUser{API: api})
		})

		router.Group(func(router chi.Router) {
			router.Use(ihttp.Session(logger, sessionManager, sessionExpiration))

			router.Method(http.MethodPost, "/user/logout", LogoutUser{API: api})
			router.Method(http.MethodPost, "/user/logout-all", LogoutAllUser{API: api})
			router.Method(http.MethodPost, "/user/update-password", UpdateUserPassword{API: api})
			router.Method(http.MethodPost, "/user/resend-verification-email", ResendEmailVerification{API: api})
			router.Method(http.MethodGet, "/user", User{API: api})
		})
	})

	return &api
}

type API struct {
	Mux *chi.Mux

	logger         *zap.Logger
	ctrl           IController
	sessionManager ihttp.ISessionManager

	cookieOptions ihttp.CookieOptions
}
