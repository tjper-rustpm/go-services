package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	rpmerrors "github.com/tjper/rustcron/cmd/user/errors"
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

type ISessionManager interface {
	CreateSession(context.Context, session.Session, time.Duration) error
	DeleteSession(context.Context, session.Session) error
	InvalidateUserSessionsBefore(context.Context, fmt.Stringer, time.Time) error
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
	sessionManager ISessionManager,
	store IStore,
	emailer IEmailer,
	admins IAdminSet,
	activeSessionExpiration time.Duration,
	absoluteSessionExpiration time.Duration,
) *Controller {
	return &Controller{
		sessionManager:            sessionManager,
		store:                     store,
		emailer:                   emailer,
		admins:                    admins,
		activeSessionExpiration:   activeSessionExpiration,
		absoluteSessionExpiration: absoluteSessionExpiration,
	}
}

// Controller is responsible for interactions with user resources. All
// interactions with the user resources occur through the Controller.
type Controller struct {
	sessionManager ISessionManager
	store          IStore
	emailer        IEmailer
	admins         IAdminSet

	activeSessionExpiration   time.Duration
	absoluteSessionExpiration time.Duration
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
	if !isEmail(input.Email) {
		return nil, rpmerrors.EmailError("unknown characters")
	}
	if err := validatePassword(input.Password); err != nil {
		return nil, err
	}
	_, err := ctrl.store.UserByEmail(ctx, input.Email)
	if err == nil {
		return nil, rpmerrors.EmailError("invalid; please use another email")
	}
	if err != nil && !errors.Is(err, rpmerrors.UserDNE) {
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
	if err := validatePassword(input.CurrentPassword); err != nil {
		return nil, err
	}
	if err := validatePassword(input.NewPassword); err != nil {
		return nil, err
	}

	user, err := ctrl.store.User(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(
		user.Password,
		hash([]byte(input.CurrentPassword), []byte(user.Salt)),
	) {
		return nil, rpmerrors.AuthError("invalid credentials")
	}

	password := hash([]byte(input.NewPassword), []byte(user.Salt))
	return ctrl.store.UpdateUserPassword(ctx, input.ID, password)
}

// LoginUserInput is the input for the Controller.LoginUser method.
type LoginUserInput struct {
	Email    string
	Password string
}

// LoginUserOutput is the output for the Controller.LoginUser method.
type LoginUserOutput struct {
	User      *model.User
	SessionID string
}

// LoginUser ensures the passed credentials are valid. On success, the
// logged-in user is returned to the caller, and a the user's new session ID.
func (ctrl Controller) LoginUser(
	ctx context.Context,
	input LoginUserInput,
) (*LoginUserOutput, error) {
	if !isEmail(input.Email) {
		return nil, rpmerrors.EmailError("unknown characters")
	}
	if err := validatePassword(input.Password); err != nil {
		return nil, err
	}

	user, err := ctrl.store.UserByEmail(ctx, input.Email)
	if errors.Is(err, rpmerrors.UserDNE) {
		return nil, rpmerrors.AuthError("invalid credentials")
	}
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(
		user.Password,
		hash([]byte(input.Password), []byte(user.Salt)),
	) {
		return nil, rpmerrors.AuthError("invalid credentials")
	}

	sessionID, err := rand.GenerateString(32)
	if err != nil {
		return nil, err
	}
	if err := ctrl.sessionManager.CreateSession(
		ctx,
		session.Session{
			ID:                 sessionID,
			User:               user.ToSessionUser(),
			LastActivityAt:     time.Now(),
			AbsoluteExpiration: time.Now().Add(ctrl.absoluteSessionExpiration),
			CreatedAt:          time.Now(),
		},
		ctrl.activeSessionExpiration,
	); err != nil {
		return nil, err
	}

	return &LoginUserOutput{
		User:      user,
		SessionID: sessionID,
	}, nil
}

// LogoutUserSession invalidates the passed session, resulting in any user
// using the session to be logged out.
func (ctrl Controller) LogoutUserSession(ctx context.Context, sess session.Session) error {
	return ctrl.sessionManager.DeleteSession(ctx, sess)
}

// LogoutAllUserSessions invalidates all existing sessions related to the user,
// resulting in any user using these sessions to be logged out.
func (ctrl Controller) LogoutAllUserSessions(ctx context.Context, userID fmt.Stringer) error {
	return ctrl.sessionManager.InvalidateUserSessionsBefore(ctx, userID, time.Now())
}

// User retrieves the user associated with the passed ID.
func (ctrl Controller) User(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return ctrl.store.User(ctx, id)
}

// VerifyEmail verifies the email associated with passed hash. The hash is a
// cryptographically-secure pseudorandom number.
func (ctrl Controller) VerifyEmail(ctx context.Context, hash string) (*model.User, error) {
	user, err := ctrl.store.UserByVerificationHash(ctx, hash)
	if errors.Is(err, rpmerrors.UserDNE) {
		return nil, rpmerrors.HashError("invalid hash")
	}
	if err != nil {
		return nil, err
	}
	if user.IsVerified() {
		return nil, rpmerrors.HashError("already verified")
	}
	if user.IsVerificationHashStale() {
		return nil, rpmerrors.AuthError("invalid credentials")
	}
	if user.VerificationHash != hash {
		return nil, rpmerrors.AuthError("invalid credentials")
	}
	return ctrl.store.VerifyEmail(ctx, user.ID, hash)
}

// ValidateResetPasswordHash
func (ctrl Controller) ValidateResetPasswordHash(
	ctx context.Context,
	hash string,
) error {
	reset, err := ctrl.store.PasswordResetByHash(ctx, hash)
	if errors.Is(err, rpmerrors.PasswordResetDNE) {
		return rpmerrors.AuthError("invalid credentials")
	}
	if err != nil {
		return err
	}
	if reset.IsRequestStale() {
		return rpmerrors.AuthError("stale hash")
	}
	return nil
}

// ResendEmailVerification resends the "verify email" email and resets related
// data such as time sent, etc.
func (ctrl Controller) ResendEmailVerification(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := ctrl.store.User(ctx, id)
	if err != nil {
		return nil, err
	}
	if user.IsVerified() {
		return nil, rpmerrors.EmailAlreadyVerified
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
	if !isEmail(email) {
		return rpmerrors.EmailError("unknown characters")
	}

	_, err := ctrl.store.UserByEmail(ctx, email)
	if errors.Is(err, rpmerrors.UserDNE) {
		return rpmerrors.EmailAddressNotRecognized
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

	// ErrPasswordResetStale indicates that a password reset request has expired,
	// and the related hash is no longer valid.
	ErrPasswordResetRequestStale = errors.New("password reset request stale")
)

// ResetPassword sets the password of the user associated with the passed hash
// to the specified password.
func (ctrl Controller) ResetPassword(
	ctx context.Context,
	resetPasswordHash string,
	password string,
) error {
	if err := validatePassword(password); err != nil {
		return err
	}

	user, err := ctrl.store.UserByResetPasswordHash(ctx, resetPasswordHash)
	if errors.Is(err, rpmerrors.UserDNE) {
		return rpmerrors.AuthError("invalid credentials")
	}
	if err != nil {
		return err
	}

	reset, err := ctrl.store.PasswordResetByHash(ctx, resetPasswordHash)
	if errors.Is(err, rpmerrors.PasswordResetDNE) {
		return rpmerrors.AuthError("invalid credentials")
	}
	if err != nil {
		return err
	}
	if reset.IsRequestStale() {
		return rpmerrors.AuthError("stale hash")
	}
	if reset.IsCompleted() {
		return rpmerrors.AuthError("reset previously completed")
	}

	if err := ctrl.store.CompleteUserPasswordReset(
		ctx,
		user.ID,
		reset.ID,
		hash([]byte(password), []byte(user.Salt)),
	); err != nil {
		return err
	}

	return ctrl.LogoutAllUserSessions(ctx, user.ID)
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

var (
	emailRE = regexp.MustCompile(`^[a-zA-Z0-9_+&*-]+(?:\.[a-zA-Z0-9_+&*-]+)*@(?:[a-zA-Z0-9-]+\.)+[a-zA-Z]{2,7}$`)

	passwordRE            = regexp.MustCompile(`^[a-zA-Z\d \!\"\#\$\%\&\'\(\)\*\+\,\-\.\/\:\;\<\=\>\?\@\[\]\^\_\x60\{\|\}\~]{8,64}$`)
	atLeastOneLowerCaseRE = regexp.MustCompile(`[a-z]+`)
	atLeastOneUpperCaseRE = regexp.MustCompile(`[A-Z]+`)
	atLeastOneNumberRE    = regexp.MustCompile(`[\d]+`)
)

func isEmail(s string) bool {
	return emailRE.MatchString(s)
}

// validatePassword matches against strings that satisfy the following
// requirements:
// - between 8 and 64 characters in length
// - at least one lower-case letter
// - at least one upper-case letter
// - at least one number
// - special characters are allowed
// In the event there is not a match, the reason is returned as an error.
func validatePassword(s string) error {
	const (
		minLength = 8
		maxLength = 64
	)
	switch {
	case len(s) < minLength:
		return rpmerrors.PasswordError(fmt.Sprintf("minimum of %d characters", minLength))
	case len(s) > maxLength:
		return rpmerrors.PasswordError(fmt.Sprintf("maximum of %d characters", maxLength))
	case !passwordRE.MatchString(s):
		return rpmerrors.PasswordError("unknown characters")
	case !atLeastOneLowerCaseRE.MatchString(s):
		return rpmerrors.PasswordError("at least one lower-case letter required")
	case !atLeastOneUpperCaseRE.MatchString(s):
		return rpmerrors.PasswordError("at least one upper-case letter required")
	case !atLeastOneNumberRE.MatchString(s):
		return rpmerrors.PasswordError("at least one number required")
	}
	return nil
}
