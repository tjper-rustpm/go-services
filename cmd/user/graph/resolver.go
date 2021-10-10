//go:generate go run github.com/99designs/gqlgen
package graph

import (
	"context"
	"net/http"

	"github.com/tjper/rustcron/cmd/user/controller"
	"github.com/tjper/rustcron/cmd/user/model"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// IController represents the API by which the resolver may control the user
// data model.
type IController interface {
	CreateUser(context.Context, controller.CreateUserInput) (*model.User, error)
	User(context.Context, uuid.UUID) (*model.User, error)
	UpdateUserPassword(context.Context, controller.UpdateUserPasswordInput) (*model.User, error)

	LoginUser(context.Context, controller.LoginUserInput) (*controller.LoginUserOutput, error)
	LogoutUser(context.Context, string) error

	RequestPasswordReset(context.Context, string) error
	ResetPassword(context.Context, string, string) error
	ResendEmailVerification(context.Context, uuid.UUID) (*model.User, error)
}

func NewResolver(
	logger *zap.Logger,
	ctrl IController,
	cookieDomain string,
	cookieSecure bool,
	cookieSameSite http.SameSite,
) *Resolver {
	return &Resolver{
		logger:         logger,
		ctrl:           ctrl,
		cookieDomain:   cookieDomain,
		cookieSecure:   cookieSecure,
		cookieSameSite: cookieSameSite,
	}
}

// Resolver resolves graphql queries and mutations.
type Resolver struct {
	logger *zap.Logger
	ctrl   IController

	cookieDomain   string
	cookieSecure   bool
	cookieSameSite http.SameSite
}
