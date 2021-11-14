package rest

import (
	"context"
	errors "errors"
	http "net/http"

	"github.com/tjper/rustcron/cmd/user/controller"
	uerrors "github.com/tjper/rustcron/cmd/user/errors"
	"github.com/tjper/rustcron/cmd/user/model"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"

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
	cookieOptions ihttp.CookieOptions,
) *API {
	return &API{
		logger:        logger,
		ctrl:          ctrl,
		cookieOptions: cookieOptions,
	}
}

type API struct {
	logger *zap.Logger
	ctrl   IController

	cookieOptions ihttp.CookieOptions
}

func (api API) CreateUser() http.HandlerFunc {
	type body struct {
		Email    string
		Password string
	}

	return func(w http.ResponseWriter, req *http.Request) {
		var b body
		if err := api.read(w, req, b); err != nil {
			return
		}

		user, err := api.ctrl.CreateUser(
			req.Context(),
			controller.CreateUserInput{Email: b.Email, Password: b.Password},
		)
		if emailErr := uerrors.AsEmailError(err); emailErr != nil {
			http.Error(w, "invalid email", http.StatusBadRequest)
			return
		}
		if passwordErr := uerrors.AsPasswordError(err); passwordErr != nil {
			http.Error(w, "invalid password", http.StatusBadRequest)
			return
		}
		if errors.Is(err, uerrors.EmailAlreadyInUse) {
			http.Error(w, "invalid email", http.StatusConflict)
			return
		}
		if err != nil {
			ihttp.ErrInternal(w)
			return
		}

		api.write(w, http.StatusCreated, user)
	}
}

func (api API) UpdateUserPassword() http.HandlerFunc {
	type body struct {
		UserID          uuid.UUID
		CurrentPassword string
		NewPassword     string
	}

	return func(w http.ResponseWriter, req *http.Request) {
		var b body
		if err := api.read(w, req, b); err != nil {
			return
		}

		sess, ok := session.FromContext(req.Context())
		if !ok {
			ihttp.ErrUnauthorized(w)
			return
		}
		if !sess.IsAuthorized(b.UserID) {
			ihttp.ErrForbidden(w)
			return
		}

		user, err := api.ctrl.UpdateUserPassword(
			req.Context(),
			controller.UpdateUserPasswordInput{
				ID:              b.UserID,
				CurrentPassword: b.CurrentPassword,
				NewPassword:     b.NewPassword,
			},
		)
		if passwordErr := uerrors.AsPasswordError(err); passwordErr != nil {
			ihttp.ErrBadRequest(w, "password")
			return
		}
		if authErr := uerrors.AsAuthError(err); authErr != nil {
			ihttp.ErrForbidden(w)
			return
		}
		if err != nil {
			ihttp.ErrInternal(w)
			return
		}

		if err := api.ctrl.LogoutUser(req.Context(), sess.ID); err != nil {
			ihttp.ErrInternal(w)
			return
		}

		api.write(w, http.StatusCreated, user)
	}
}

func (api API) VerifyEmail() http.HandlerFunc {
	type body struct {
		Hash string
	}

	return func(w http.ResponseWriter, req *http.Request) {
		var b body
		if err := api.read(w, req, b); err != nil {
			return
		}

		_, err := api.ctrl.VerifyEmail(req.Context(), b.Hash)
		if authErr := uerrors.AsAuthError(err); authErr != nil {
			ihttp.ErrForbidden(w)
			return
		}
		if hashErr := uerrors.AsHashError(err); hashErr != nil {
			ihttp.ErrForbidden(w)
			return
		}
		if err != nil {
			ihttp.ErrInternal(w)
			return
		}

		api.write(w, http.StatusCreated, nil)
	}
}

func (api API) LoginUser() http.HandlerFunc {
	type body struct {
		Email    string
		Password string
	}

	return func(w http.ResponseWriter, req *http.Request) {
		var b body
		if err := api.read(w, req, b); err != nil {
			return
		}

		user, err := api.ctrl.LoginUser(
			req.Context(),
			controller.LoginUserInput{Email: b.Email, Password: b.Password},
		)
		if authErr := uerrors.AsAuthError(err); authErr != nil {
			ihttp.ErrUnauthorized(w)
			return
		}
		if err != nil {
			ihttp.ErrInternal(w)
			return
		}

		ihttp.SetSessionCookie(
			w,
			user.SessionID,
			api.cookieOptions,
		)

		api.write(w, http.StatusCreated, user)
	}
}

func (api API) LogoutUser() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		sess, ok := api.session(req.Context(), w)
		if !ok {
			return
		}

		if err := api.ctrl.LogoutUser(req.Context(), sess.ID); err != nil {
			ihttp.ErrInternal(w)
			return
		}

		api.write(w, http.StatusCreated, nil)
	}
}

func (api API) ForgotPassword() http.HandlerFunc {
	type body struct {
		Email string
	}

	return func(w http.ResponseWriter, req *http.Request) {
		var b body
		if err := api.read(w, req, b); err != nil {
			return
		}

		err := api.ctrl.RequestPasswordReset(req.Context(), b.Email)
		if emailErr := uerrors.AsEmailError(err); emailErr != nil {
			ihttp.ErrBadRequest(w, "email")
			return
		}
		if errors.Is(err, uerrors.EmailAddressNotRecognized) {
			// Response is independent of whether an email address is found. This is
			// to prevent attackers from determining which email addresses are
			// associated with users.
			api.write(w, http.StatusCreated, nil)
			return
		}
		if err != nil {
			ihttp.ErrInternal(w)
			return
		}

		api.write(w, http.StatusCreated, nil)
	}
}

func (api API) ChangePassword() http.HandlerFunc {
	type body struct {
		Hash     string
		Password string
	}

	return func(w http.ResponseWriter, req *http.Request) {
		var b body
		if err := api.read(w, req, b); err != nil {
			return
		}

		err := api.ctrl.ResetPassword(req.Context(), b.Hash, b.Password)
		if errors.Is(err, controller.ErrResetHashNotRecognized) ||
			errors.Is(err, controller.ErrPasswordResetRequestStale) {
			ihttp.ErrForbidden(w)
			return
		}
		if err != nil {
			ihttp.ErrInternal(w)
			return
		}

		api.write(w, http.StatusCreated, nil)
	}
}

func (api API) ResendEmailVerification() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		sess, ok := api.session(req.Context(), w)
		if !ok {
			return
		}

		_, err := api.ctrl.ResendEmailVerification(req.Context(), sess.User.ID)
		if errors.Is(err, uerrors.EmailAlreadyVerified) {
			ihttp.ErrConflict(w)
			return
		}
		if err != nil {
			ihttp.ErrInternal(w)
			return
		}

		api.write(w, http.StatusCreated, nil)
	}
}

func (api API) Me() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		sess, ok := api.session(req.Context(), w)
		if !ok {
			return
		}

		user, err := api.ctrl.User(req.Context(), sess.User.ID)
		if err != nil {
			ihttp.ErrInternal(w)
			return
		}

		api.write(w, http.StatusCreated, user)
	}
}
