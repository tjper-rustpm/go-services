package rest

import (
	"context"
	"fmt"
	"net/http"

	rpmerrors "github.com/tjper/rustcron/cmd/user/errors"
	"github.com/tjper/rustcron/cmd/user/model"

	"go.uber.org/zap"
)

type IController interface {
	VerifyEmail(context.Context, string) (*model.User, error)
	ValidateResetPasswordHash(context.Context, string) error
}

func NewEndpoints(logger *zap.Logger, ctrl IController) *Endpoints {
	return &Endpoints{
		logger: logger,
		ctrl:   ctrl,
	}
}

type Endpoints struct {
	logger *zap.Logger
	ctrl   IController
}

func (endpoints Endpoints) ValidateResetPasswordHashHandler(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	if len(hash) == 0 {
		endpoints.logger.Error("error retrieving hash from URL query parameter")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err := endpoints.ctrl.ValidateResetPasswordHash(r.Context(), hash)
	if authErr := rpmerrors.AsAuthError(err); authErr != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if err != nil {
		endpoints.logger.Error("error validating reset password hash", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	http.Redirect(
		w,
		nil,
		fmt.Sprintf("/graphql?hash=%s", hash),
		http.StatusFound,
	)
}
