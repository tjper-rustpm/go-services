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
	"github.com/tjper/rustcron/internal/validator"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	validatorv10 "github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ISessionManager interface {
	CreateSession(context.Context, session.Session) error
	RetrieveSession(context.Context, string) (*session.Session, error)
	UpdateSession(context.Context, string, func(*session.Session)) (*session.Session, error)
	TouchSession(context.Context, string) (*session.Session, error)
	DeleteSession(context.Context, session.Session) error
	InvalidateUserSessionsBefore(context.Context, fmt.Stringer, time.Time) error
}

type IController interface {
	CreateUser(context.Context, controller.CreateUserInput) (*model.User, error)
	User(context.Context, uuid.UUID) (*model.User, error)
	UpdateUserPassword(context.Context, controller.UpdateUserPasswordInput) (*model.User, error)

	LoginUser(context.Context, controller.LoginUserInput) (*model.User, error)

	VerifyEmail(context.Context, string) (*model.User, error)
	RequestPasswordReset(context.Context, string) error
	ResetPassword(context.Context, string, string) (*model.User, error)
	ResendEmailVerification(context.Context, uuid.UUID) (*model.User, error)
}

func NewAPI(
	logger *zap.Logger,
	ctrl IController,
	cookieOptions ihttp.CookieOptions,
	sessionManager ISessionManager,
	sessionMiddleware *ihttp.SessionMiddleware,
	healthz http.Handler,
) *API {
	api := API{
		Mux:            chi.NewRouter(),
		logger:         logger,
		valid:          validator.New(),
		ctrl:           ctrl,
		cookieOptions:  cookieOptions,
		sessionManager: sessionManager,
	}

	api.Mux.Handle("/healthz", healthz)

	api.Mux.Route("/v1", func(router chi.Router) {
		router.Use(
			sessionMiddleware.InjectSessionIntoCtx(),
			sessionMiddleware.Touch(),
			middleware.RequestLogger(ihttp.NewZapLogFormatter(logger)),
		)

		router.Method(http.MethodGet, "/user/session", Session{API: api})
		router.Method(http.MethodPost, "/user/forgot-password", ForgotPassword{API: api})
		router.Method(http.MethodPost, "/user/verify-email", VerifyEmail{API: api})
		router.Method(http.MethodPost, "/user/reset-password", ResetPassword{API: api})

		router.Group(func(router chi.Router) {
			router.Method(http.MethodPost, "/user", CreateUser{API: api})
			router.Method(http.MethodPost, "/user/login", LoginUser{API: api})
		})

		router.Group(func(router chi.Router) {
			router.Use(sessionMiddleware.IsAuthenticated())

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
	valid          *validatorv10.Validate
	ctrl           IController
	sessionManager ISessionManager

	cookieOptions ihttp.CookieOptions
}
