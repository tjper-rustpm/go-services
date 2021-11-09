package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/user/controller"
	rpmerrors "github.com/tjper/rustcron/cmd/user/errors"
	"github.com/tjper/rustcron/cmd/user/graph/generated"
	"github.com/tjper/rustcron/cmd/user/graph/model"
	gerrors "github.com/tjper/rustcron/internal/graph/errors"
	"github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
	"go.uber.org/zap"
)

func (r *mutationResolver) CreateUser(ctx context.Context, input model.CreateUserInput) (*model.CreateUserResult, error) {
	user, err := r.ctrl.CreateUser(
		ctx,
		controller.CreateUserInput{
			Email:    input.Email,
			Password: input.Password,
		},
	)
	if emailErr := rpmerrors.AsEmailError(err); emailErr != nil {
		return nil, emailErr
	}
	if passwordErr := rpmerrors.AsPasswordError(err); passwordErr != nil {
		return nil, passwordErr
	}
	if errors.Is(err, rpmerrors.EmailAlreadyInUse) {
		return nil, err
	}
	if err != nil {
		r.logger.Error("error creating user", zap.Error(err))
		return nil, gerrors.ErrInternalServer
	}

	return &model.CreateUserResult{
		User: toModelUser(*user),
	}, nil
}

func (r *mutationResolver) UpdateUserPassword(ctx context.Context, input model.UpdateUserPasswordInput) (*model.UpdateUserPasswordResult, error) {
	sess, ok := session.FromContext(ctx)
	if !ok {
		r.logger.Error("session no longer exists for request's user")
		return nil, gerrors.ErrUnauthenticated
	}
	if !sess.IsAuthorized(input.ID) {
		r.logger.Error("client is not authorized to access this resource")
		return nil, gerrors.ErrUnauthorized
	}

	userID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, gerrors.ErrInvalidUUID
	}

	user, err := r.ctrl.UpdateUserPassword(
		ctx,
		controller.UpdateUserPasswordInput{
			ID:              userID,
			CurrentPassword: input.CurrentPassword,
			NewPassword:     input.NewPassword,
		},
	)
	if passwordErr := rpmerrors.AsPasswordError(err); passwordErr != nil {
		return nil, passwordErr
	}
	if authErr := rpmerrors.AsAuthError(err); authErr != nil {
		return nil, authErr
	}
	if err != nil {
		r.logger.Error("error updating user password", zap.Error(err))
		return nil, gerrors.ErrInternalServer
	}

	if err := r.ctrl.LogoutUser(ctx, sess.ID); err != nil {
		r.logger.Error("error logging-out user", zap.Error(err))
		return nil, gerrors.ErrInternalServer
	}

	return &model.UpdateUserPasswordResult{
		User: toModelUser(*user),
	}, nil
}

func (r *mutationResolver) VerifyEmail(ctx context.Context, input model.VerifyEmailInput) (*model.VerifyEmailResult, error) {
	user, err := r.ctrl.VerifyEmail(ctx, input.Hash)
	if authErr := rpmerrors.AsAuthError(err); authErr != nil {
		r.logger.Error("error verifying email", zap.Error(authErr))
		return nil, gerrors.ErrUnauthorized
	}
	if hashErr := rpmerrors.AsHashError(err); hashErr != nil {
		r.logger.Error("error verifying email", zap.Error(hashErr))
		return nil, gerrors.ErrUnauthorized
	}
	if err != nil {
		r.logger.Error("error verifying email", zap.Error(err))
		return nil, gerrors.ErrInternalServer
	}

	return &model.VerifyEmailResult{
		User: toModelUser(*user),
	}, nil
}

func (r *mutationResolver) LoginUser(ctx context.Context, input model.LoginUserInput) (*model.LoginUserResult, error) {
	out, err := r.ctrl.LoginUser(
		ctx,
		controller.LoginUserInput{Email: input.Email, Password: input.Password},
	)
	if authErr := rpmerrors.AsAuthError(err); authErr != nil {
		return nil, authErr
	}
	if err != nil {
		r.logger.Error("error logging-in user", zap.Error(err))
		return nil, gerrors.ErrInternalServer
	}
	access, ok := http.AccessFromContext(ctx)
	if !ok {
		r.logger.Error("error accessing cookies", zap.Error(gerrors.ErrInternalServer))
		return nil, gerrors.ErrInternalServer
	}
	access.SetSessionID(
		out.SessionID,
		r.cookieDomain,
		r.cookieSecure,
		r.cookieSameSite,
	)

	return &model.LoginUserResult{
		User: toModelUser(*out.User),
	}, nil
}

func (r *mutationResolver) LogoutUser(ctx context.Context) (bool, error) {
	sess, ok := session.FromContext(ctx)
	if !ok {
		r.logger.Error("session no longer exists for request's user", zap.Error(gerrors.ErrUnauthenticated))
		return false, gerrors.ErrUnauthenticated
	}

	if err := r.ctrl.LogoutUser(ctx, sess.ID); err != nil {
		r.logger.Error("error logging-out user", zap.Error(err))
		return false, gerrors.ErrInternalServer
	}
	return true, nil
}

func (r *mutationResolver) ForgotPassword(ctx context.Context, input model.ForgotPasswordInput) (bool, error) {
	err := r.ctrl.RequestPasswordReset(ctx, input.Email)
	if emailErr := rpmerrors.AsEmailError(err); emailErr != nil {
		return false, emailErr
	}
	if errors.Is(err, rpmerrors.EmailAddressNotRecognized) {
		// Response is independent of whether an email address is found. This is
		// to prevent attackers from determining which email addresses are
		// associated with users.
		return true, nil
	}
	if err != nil {
		r.logger.Error("error requesting password-reset", zap.Error(err))
		return false, gerrors.ErrInternalServer
	}
	return true, nil
}

func (r *mutationResolver) ChangePassword(ctx context.Context, input model.ChangePasswordInput) (bool, error) {
	err := r.ctrl.ResetPassword(ctx, input.Hash, input.Password)
	if errors.Is(err, controller.ErrResetHashNotRecognized) {
		return false, gerrors.ErrUnauthorized
	}
	if errors.Is(err, controller.ErrPasswordResetRequestStale) {
		return false, gerrors.ErrResetWindowExpired
	}
	if err != nil {
		r.logger.Error("error reseting user password", zap.Error(err))
		return false, gerrors.ErrInternalServer
	}
	return true, nil
}

func (r *mutationResolver) ResendEmailVerification(ctx context.Context) (bool, error) {
	sess, ok := session.FromContext(ctx)
	if !ok {
		r.logger.Error("session no longer exists for request's user")
		return false, gerrors.ErrUnauthenticated
	}

	_, err := r.ctrl.ResendEmailVerification(ctx, sess.User.ID)
	if errors.Is(err, rpmerrors.EmailAlreadyVerified) {
		return false, err
	}
	if err != nil {
		r.logger.Error("error resending email verification", zap.Error(err))
		return false, gerrors.ErrInternalServer
	}
	return true, nil
}

func (r *queryResolver) Me(ctx context.Context) (*model.User, error) {
	sess, ok := session.FromContext(ctx)
	if !ok {
		r.logger.Error("session no longer exists for request's user")
		return nil, gerrors.ErrUnauthenticated
	}

	user, err := r.ctrl.User(ctx, sess.User.ID)
	if err != nil {
		r.logger.Error("error retrieving user", zap.Error(err))
		return nil, gerrors.ErrInternalServer
	}
	return toModelUser(*user), nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
