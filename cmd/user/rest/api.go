package rest

import (
	"context"
	http "net/http"

	"github.com/go-chi/chi"
	"github.com/tjper/rustcron/cmd/user/controller"
	"github.com/tjper/rustcron/cmd/user/model"
	ihttp "github.com/tjper/rustcron/internal/http"

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

type IEndpoint interface {
	http.Handler
	Route(chi.Router)
}

func NewAPI(
	logger *zap.Logger,
	ctrl IController,
	cookieOptions ihttp.CookieOptions,
) *API {
	api := API{
		Mux:           chi.NewRouter(),
		logger:        logger,
		ctrl:          ctrl,
		cookieOptions: cookieOptions,
	}

	v1 := []IEndpoint{
		CreateUser{API: api},
		LoginUser{API: api},
		LogoutUser{API: api},
		Me{API: api},
		UpdateUserPassword{API: api},
		ForgotPassword{API: api},
		ChangePassword{API: api},
		ResendEmailVerification{API: api},
		VerifyEmail{API: api},
	}

	api.Mux.Route("/v1", func(router chi.Router) {
		for _, endpoint := range v1 {
			endpoint.Route(router)
		}
	})

	return &api
}

type API struct {
	Mux *chi.Mux

	logger *zap.Logger
	ctrl   IController

	cookieOptions ihttp.CookieOptions
}
