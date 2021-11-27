package rest

import (
	"context"
	http "net/http"

	"github.com/tjper/rustcron/cmd/user/controller"
	"github.com/tjper/rustcron/cmd/user/model"
	ihttp "github.com/tjper/rustcron/internal/http"

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
	LogoutUser(context.Context, string) error

	VerifyEmail(context.Context, string) (*model.User, error)
	RequestPasswordReset(context.Context, string) error
	ResetPassword(context.Context, string, string) error
	ResendEmailVerification(context.Context, uuid.UUID) (*model.User, error)
}

func NewAPI(
	logger *zap.Logger,
	ctrl IController,
	retriever ihttp.Retriever,
	cookieOptions ihttp.CookieOptions,
) *API {
	api := API{
		Mux:           chi.NewRouter(),
		logger:        logger,
		ctrl:          ctrl,
		retriever:     retriever,
		cookieOptions: cookieOptions,
	}

	api.Mux.Use(
		middleware.RequestLogger(ihttp.NewZapLogFormatter(logger)),
	)

	api.Mux.Route("/v1", func(router chi.Router) {
		router.Method(http.MethodPost, "/user", CreateUser{API: api})
		router.Method(http.MethodPost, "/user/login", LoginUser{API: api})
		router.Method(http.MethodPost, "/user/forgot-password", ForgotPassword{API: api})
		router.Method(http.MethodPost, "/user/verify-email", VerifyEmail{API: api})
		router.Method(http.MethodPost, "/user/change-password", ChangePassword{API: api})

		router.Group(func(router chi.Router) {
			router.Use(ihttp.SessionAuthenticated(retriever))

			router.Method(http.MethodPost, "/user/logout", LogoutUser{API: api})
			router.Method(http.MethodPost, "/user/update-password", UpdateUserPassword{API: api})
			router.Method(http.MethodPost, "/user/resend-verification-email", ResendEmailVerification{API: api})

			router.Method(http.MethodGet, "/user", Me{API: api})

		})

	})

	return &api
}

type API struct {
	Mux *chi.Mux

	logger    *zap.Logger
	ctrl      IController
	retriever ihttp.Retriever

	cookieOptions ihttp.CookieOptions
}
