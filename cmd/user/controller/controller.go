package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	usererrors "github.com/tjper/rustcron/cmd/user/errors"
	"github.com/tjper/rustcron/cmd/user/model"
	"github.com/tjper/rustcron/internal/rand"
	"github.com/tjper/rustcron/internal/session"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

type IAdminSet interface {
	Contains(string) bool
}

type IEmailer interface {
	SendPasswordReset(context.Context, string, string) error
	SendVerifyEmail(context.Context, string, string) error
}

type IStore interface {
	CreateUser(context.Context, *model.User) error
	UpdateUserPassword(context.Context, uuid.UUID, []byte) (*model.User, error)

	User(context.Context, uuid.UUID) (*model.User, error)
	UserByEmail(context.Context, string) (*model.User, error)
	UserByResetPasswordHash(context.Context, string) (*model.User, error)
	UserByVerificationHash(context.Context, string) (*model.User, error)

	VerifyEmail(context.Context, uuid.UUID, string) (*model.User, error)
	ResetEmailVerification(context.Context, uuid.UUID, string) (*model.User, error)

	CreateUserPasswordReset(context.Context, string, string) (*model.PasswordReset, error)
	PasswordResetByHash(context.Context, string) (*model.PasswordReset, error)
	CompleteUserPasswordReset(context.Context, uuid.UUID, uuid.UUID, []byte) error
}

func New(
	store IStore,
	emailer IEmailer,
	admins IAdminSet,
) *Controller {
	return &Controller{
		store:   store,
		emailer: emailer,
		admins:  admins,
	}
}

// Controller is responsible for interactions with user resources. All
// interactions with the user resources occur through the Controller.
type Controller struct {
	store   IStore
	emailer IEmailer
	admins  IAdminSet
}

// CreateUserInput is the input for the Controller.CreateUser method.
type CreateUserInput struct {
	Email    string
	Password string
}

// CreateUser creates a new model.User, and sends a verification email to the
// user's specified email account.
func (ctrl Controller) CreateUser(
	ctx context.Context,
	input CreateUserInput,
) (*model.User, error) {
	_, err := ctrl.store.UserByEmail(ctx, input.Email)
	if err == nil {
		return nil, fmt.Errorf("user by email; error: %w", usererrors.ErrEmailAlreadyInUse)
	}
	if err != nil && !errors.Is(err, usererrors.ErrUserDNE) {
		return nil, err
	}

	role := session.RoleStandard
	if ctrl.admins.Contains(input.Email) {
		role = session.RoleAdmin
	}
	verificationHash, err := rand.GenerateString(32)
	if err != nil {
		return nil, err
	}

	salt, err := rand.GenerateString(32)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:              input.Email,
		Password:           hash([]byte(input.Password), []byte(salt)),
		Salt:               salt,
		Role:               role,
		VerificationHash:   verificationHash,
		VerificationSentAt: time.Now(),
	}
	if err := ctrl.store.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	if err := ctrl.emailer.SendVerifyEmail(
		ctx,
		input.Email,
		verificationHash,
	); err != nil {
		return nil, err
	}
	// send email
	return user, nil
}

// UpdateUserPasswordInput is the input for the Controller.UpdateUserPassword
// method.
type UpdateUserPasswordInput struct {
	ID              uuid.UUID
	CurrentPassword string
	NewPassword     string
}

// UpdateUserPassword updates the user associated with the passed ID to utilize
// the specified password.
func (ctrl Controller) UpdateUserPassword(ctx context.Context, input UpdateUserPasswordInput) (*model.User, error) {
	user, err := ctrl.store.User(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if !user.IsPassword(hash([]byte(input.CurrentPassword), []byte(user.Salt))) {
		return nil, usererrors.AuthError("invalid credentials")
	}

	password := hash([]byte(input.NewPassword), []byte(user.Salt))
	return ctrl.store.UpdateUserPassword(ctx, input.ID, password)
}

// UpdateUserSteamInput is the input for the Controller.UpdateUserSteam method.
type UpdateUserSteamInput struct {
	ID      uuid.UUID
	SteamID uuid.UUID
}

// LoginUserInput is the input for the Controller.LoginUser method.
type LoginUserInput struct {
	Email    string
	Password string
}

// LoginUser ensures the passed credentials are valid. On success, the
// logged-in user is returned to the caller, and a the user's new session ID.
func (ctrl Controller) LoginUser(
	ctx context.Context,
	input LoginUserInput,
) (*model.User, error) {
	user, err := ctrl.store.UserByEmail(ctx, input.Email)
	if errors.Is(err, usererrors.ErrUserDNE) {
		return nil, usererrors.AuthError("invalid credentials")
	}
	if err != nil {
		return nil, err
	}

	if !user.IsPassword(hash([]byte(input.Password), []byte(user.Salt))) {
		return nil, usererrors.AuthError("invalid credentials")
	}

	return user, nil
}

// User retrieves the user associated with the passed ID.
func (ctrl Controller) User(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return ctrl.store.User(ctx, id)
}

// VerifyEmail verifies the email associated with passed hash. The hash is a
// cryptographically-secure pseudorandom number.
func (ctrl Controller) VerifyEmail(ctx context.Context, hash string) (*model.User, error) {
	user, err := ctrl.store.UserByVerificationHash(ctx, hash)
	if errors.Is(err, usererrors.ErrUserDNE) {
		return nil, usererrors.HashError("invalid hash")
	}
	if err != nil {
		return nil, err
	}
	if user.IsVerified() {
		return nil, usererrors.HashError("already verified")
	}
	if user.IsVerificationHashStale() {
		return nil, usererrors.AuthError("invalid credentials")
	}
	if user.VerificationHash != hash {
		return nil, usererrors.AuthError("invalid credentials")
	}
	return ctrl.store.VerifyEmail(ctx, user.ID, hash)
}

// ResendEmailVerification resends the "verify email" email and resets related
// data such as time sent, etc.
func (ctrl Controller) ResendEmailVerification(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := ctrl.store.User(ctx, id)
	if err != nil {
		return nil, err
	}
	if user.IsVerified() {
		return nil, usererrors.ErrEmailAlreadyVerified
	}

	verificationHash, err := rand.GenerateString(32)
	if err != nil {
		return nil, err
	}
	user, err = ctrl.store.ResetEmailVerification(
		ctx,
		id,
		verificationHash,
	)
	if err != nil {
		return nil, err
	}

	if err := ctrl.emailer.SendVerifyEmail(
		ctx,
		user.Email,
		verificationHash,
	); err != nil {
		return nil, err
	}
	return user, err
}

// RequestPasswordReset initiates the password reset process for the specified
// email address. If the specified email address is associated with a user,
// that email address will receive a "reset password" email.
func (ctrl Controller) RequestPasswordReset(ctx context.Context, email string) error {
	_, err := ctrl.store.UserByEmail(ctx, email)
	if errors.Is(err, usererrors.ErrUserDNE) {
		return usererrors.ErrEmailAddressNotRecognized
	}
	if err != nil {
		return err
	}

	passwordResetHash, err := rand.GenerateString(32)
	if err != nil {
		return err
	}
	if _, err := ctrl.store.CreateUserPasswordReset(
		ctx,
		email,
		passwordResetHash,
	); err != nil {
		return err
	}
	return ctrl.emailer.SendPasswordReset(ctx, email, passwordResetHash)
}

var (
	// ErrResetHashNotRecognized indicates that a password reset was attempted, but
	// the provided hash was not recognized.
	ErrResetHashNotRecognized = errors.New("password reset hash not recognized")

	// ErrPasswordResetRequestStale indicates that a password reset request has expired,
	// and the related hash is no longer valid.
	ErrPasswordResetRequestStale = errors.New("password reset request stale")
)

// ResetPassword sets the password of the user associated with the passed hash
// to the specified password.
func (ctrl Controller) ResetPassword(
	ctx context.Context,
	resetPasswordHash string,
	password string,
) (*model.User, error) {
	user, err := ctrl.store.UserByResetPasswordHash(ctx, resetPasswordHash)
	if errors.Is(err, usererrors.ErrUserDNE) {
		return nil, usererrors.AuthError("invalid credentials")
	}
	if err != nil {
		return nil, err
	}

	reset, err := ctrl.store.PasswordResetByHash(ctx, resetPasswordHash)
	if errors.Is(err, usererrors.ErrPasswordResetDNE) {
		return nil, usererrors.AuthError("invalid credentials")
	}
	if err != nil {
		return nil, err
	}
	if reset.IsRequestStale() {
		return nil, usererrors.AuthError("stale hash")
	}
	if reset.IsCompleted() {
		return nil, usererrors.AuthError("reset previously completed")
	}

	if err := ctrl.store.CompleteUserPasswordReset(
		ctx,
		user.ID,
		reset.ID,
		hash([]byte(password), []byte(user.Salt)),
	); err != nil {
		return nil, err
	}

	return user, nil
}

type AddUserVIPInput struct {
	UserID   uuid.UUID
	ServerID uuid.UUID
}

// --- helpers ---

func hash(password, salt []byte) []byte {
	const (
		minIterations = 2
		minMemory     = 64 * 1024
		threads       = 1
		keyLength     = 32
	)
	return argon2.IDKey(password, salt, minIterations, minMemory, threads, keyLength)
}
