// +build integration

package graph

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/user/config"
	"github.com/tjper/rustcron/cmd/user/controller"
	"github.com/tjper/rustcron/cmd/user/db"
	"github.com/tjper/rustcron/cmd/user/graph"
	"github.com/tjper/rustcron/cmd/user/graph/model"
	rpmgraph "github.com/tjper/rustcron/internal/graph"
	"github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := setup(t)

	t.Run("create user with invalid email", func(t *testing.T) {
		s.createUser(
			ctx,
			t,
			model.CreateUserInput{
				Email:    "invalid!@email.com",
				Password: "1ValidPassword",
				Role:     model.RoleKindStandard,
			},
			expectedCreateUserResult{
				err:    controller.ErrInvalidEmailAddress,
				result: nil,
			},
		)
	})

	t.Run("create user with invalid password", func(t *testing.T) {
		input := model.CreateUserInput{
			Email: "valid@email.com",
			Role:  model.RoleKindStandard,
		}
		exp := expectedCreateUserResult{
			err:    controller.ErrInvalidPassword,
			result: nil,
		}

		t.Run("too short password", func(t *testing.T) {
			input.Password = "6Chars"
			s.createUser(ctx, t, input, exp)
		})
		t.Run("too long password", func(t *testing.T) {
			input.Password = "TooManyChars" + strings.Repeat("1", 64)
			s.createUser(ctx, t, input, exp)
		})
		t.Run("no lower-case letter password", func(t *testing.T) {
			input.Password = "1NOLOWERCASE"
			s.createUser(ctx, t, input, exp)
		})
		t.Run("no upper-case letter password", func(t *testing.T) {
			input.Password = "1UPPERCASE"
			s.createUser(ctx, t, input, exp)
		})
		t.Run("no number password", func(t *testing.T) {
			input.Password = "NoNumber"
			s.createUser(ctx, t, input, exp)
		})
	})

	t.Run("create user with valid credentials", func(t *testing.T) {
		s.createUser(
			ctx,
			t,
			model.CreateUserInput{
				Email:    "valid@email.com",
				Password: "1ValidPassword",
				Role:     model.RoleKindStandard,
			},
			expectedCreateUserResult{
				err: nil,
				result: &model.CreateUserResult{
					User: &model.User{
						Email: "valid@email.com",
						Role:  model.RoleKindStandard,
					},
				},
			},
		)
	})

	t.Run("create user with email already being used", func(t *testing.T) {
		s.createUser(
			ctx,
			t,
			model.CreateUserInput{
				Email:    "valid@email.com",
				Password: "1ValidPassword",
				Role:     model.RoleKindStandard,
			},
			expectedCreateUserResult{
				err:    controller.ErrEmailAlreadyInUse,
				result: nil,
			},
		)
	})

	t.Run("failed login attempts", func(t *testing.T) {
		input := model.LoginUserInput{
			Email:    "valid@email.com",
			Password: "1ValidPassword",
		}
		exp := expectedLoginUserResult{
			err:    rpmgraph.ErrUnauthorized,
			result: nil,
		}

		t.Run("unrecognized email", func(t *testing.T) {
			input.Email = "unrecognized@email.com"
			s.loginUser(ctx, t, input, exp)
		})
		t.Run("incorrect password", func(t *testing.T) {
			input.Email = "valid@email.com"
			input.Password = "1IncorrectPassword"
			s.loginUser(ctx, t, input, exp)
		})
	})

	t.Run("successful login", func(t *testing.T) {
		s.loginUser(
			ctx,
			t,
			model.LoginUserInput{
				Email:    "valid@email.com",
				Password: "1ValidPassword",
			},
			expectedLoginUserResult{
				err: nil,
				result: &model.LoginUserResult{
					User: &model.User{
						Email: "valid@email.com",
						Role:  model.RoleKindStandard,
					},
				},
			},
		)
	})

	t.Run("update user password", func(t *testing.T) {
		input := model.UpdateUserPasswordInput{
			ID:          s.session.User.ID.String(),
			NewPassword: "1NewValidPassword",
		}
		exp := expectedUpdateUserPasswordResult{
			err: nil,
			result: &model.UpdateUserPasswordResult{
				User: &model.User{
					Email: "valid@email.com",
					Role:  model.RoleKindStandard,
				},
			},
		}

		t.Run("incorrect current password", func(t *testing.T) {
			input.CurrentPassword = "1IncorrectCurrentPassword"
			exp.err = rpmgraph.ErrUnauthorized
			s.updateUserPassword(ctx, t, input, exp)
		})
		t.Run("correct current password", func(t *testing.T) {
			input.CurrentPassword = "1ValidPassword"
			exp.err = nil
			s.updateUserPassword(ctx, t, input, exp)
		})
	})

	t.Run("login after user password update", func(t *testing.T) {
		input := model.LoginUserInput{
			Email: "valid@email.com",
		}
		exp := expectedLoginUserResult{
			err: nil,
			result: &model.LoginUserResult{
				User: &model.User{
					Email: "valid@email.com",
					Role:  model.RoleKindStandard,
				},
			},
		}

		t.Run("incorrect password", func(t *testing.T) {
			input.Password = "1IncorrectPassword"
			exp.err = rpmgraph.ErrUnauthorized
			s.loginUser(ctx, t, input, exp)
		})

		t.Run("correct password", func(t *testing.T) {
			input.Password = "1NewValidPassword"
			exp.err = nil
			s.loginUser(ctx, t, input, exp)
		})
	})

	t.Run("resend email verification", func(t *testing.T) {
		s.resendEmailVerification(
			ctx,
			t,
			model.ResendEmailVerificationInput{
				ID: s.session.User.ID.String(),
			},
			expectedResendEmailVerificationResult{
				err:    nil,
				result: true,
			},
		)
	})

	t.Run("logout", func(t *testing.T) {
		s.logoutUser(
			ctx,
			t,
			expectedLogoutUserResult{
				err:    nil,
				result: true,
			},
		)
	})

	t.Run("forgot password", func(t *testing.T) {
		input := model.ForgotPasswordInput{}
		exp := expectedForgotPasswordResult{
			err:    nil,
			result: true,
			addr:   "valid@email.com",
		}

		t.Run("unrecognized email", func(t *testing.T) {
			input.Email = "unrecognized@email.com"
			exp.addrExists = false
			s.forgotPassword(ctx, t, input, exp)
		})
		t.Run("recognized email", func(t *testing.T) {
			input.Email = "valid@email.com"
			exp.addrExists = true
			s.forgotPassword(ctx, t, input, exp)
		})
	})

	t.Run("change password", func(t *testing.T) {
		input := model.ChangePasswordInput{
			Password: "1ChangedPassword",
		}
		exp := expectedChangePasswordResult{
			err:    nil,
			result: true,
		}

		t.Run("incorrect hash", func(t *testing.T) {
			input.Hash = "incorrect-hash"
			exp.err = rpmgraph.ErrUnauthorized
			exp.result = false
			s.changePassword(ctx, t, input, exp)
		})
		t.Run("correct hash", func(t *testing.T) {
			input.Hash = s.resetPasswordHash
			exp.err = nil
			exp.result = true
			s.changePassword(ctx, t, input, exp)
		})
	})

	t.Run("login after password change", func(t *testing.T) {
		input := model.LoginUserInput{
			Email: "valid@email.com",
		}
		exp := expectedLoginUserResult{
			err: nil,
			result: &model.LoginUserResult{
				User: &model.User{
					Email: "valid@email.com",
					Role:  model.RoleKindStandard,
				},
			},
		}

		t.Run("incorrect password", func(t *testing.T) {
			input.Password = "1NewValidPassword"
			exp.err = rpmgraph.ErrUnauthorized
			s.loginUser(ctx, t, input, exp)
		})

		t.Run("correct password", func(t *testing.T) {
			input.Password = "1ChangedPassword"
			exp.err = nil
			s.loginUser(ctx, t, input, exp)
		})
	})
}

type suite struct {
	resolver *graph.Resolver
	emailer  *emailer

	session           *session.Session
	resetPasswordHash string
}

type expectedCreateUserResult struct {
	err    error
	result *model.CreateUserResult
}

func (s suite) createUser(
	ctx context.Context,
	t *testing.T,
	input model.CreateUserInput,
	exp expectedCreateUserResult,
) {
	result, err := s.resolver.Mutation().CreateUser(ctx, input)
	assert.ErrorIs(t, err, exp.err)
	if err != nil {
		return
	}
	result.User.Scrub()
	assert.Equal(t, exp.result, result)
}

type expectedLoginUserResult struct {
	err    error
	result *model.LoginUserResult
}

func (s *suite) loginUser(
	ctx context.Context,
	t *testing.T,
	input model.LoginUserInput,
	exp expectedLoginUserResult,
) {
	r := httptest.NewRequest("POST", "http://localhost", nil)
	w := httptest.NewRecorder()
	ctx = http.WithAccess(ctx, http.NewAccess(w, r))

	result, err := s.resolver.Mutation().LoginUser(ctx, input)
	assert.ErrorIs(t, err, exp.err)
	if err != nil {
		return
	}
	clone := result.Clone()
	clone.Scrub()
	assert.Equal(t, *exp.result, clone)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Len(t, resp.Cookies(), 1)
	s.session = &session.Session{
		ID:             resp.Cookies()[0].Value,
		LastActivityAt: time.Now(),
		User: session.User{
			ID:         uuid.MustParse(result.User.ID),
			Email:      result.User.Email,
			Role:       session.RoleKind(result.User.Role),
			VerifiedAt: result.User.VerifiedAt,
			UpdatedAt:  result.User.UpdatedAt,
			CreatedAt:  result.User.CreatedAt,
		},
	}
}

type expectedLogoutUserResult struct {
	err    error
	result bool
}

func (s *suite) logoutUser(
	ctx context.Context,
	t *testing.T,
	exp expectedLogoutUserResult,
) {
	ctx = session.WithSession(ctx, s.session)

	result, err := s.resolver.Mutation().LogoutUser(ctx)
	assert.ErrorIs(t, err, exp.err)
	if err != nil {
		return
	}
	assert.Equal(t, exp.result, result)
	s.session = nil
}

type expectedUpdateUserPasswordResult struct {
	err    error
	result *model.UpdateUserPasswordResult
}

func (s suite) updateUserPassword(
	ctx context.Context,
	t *testing.T,
	input model.UpdateUserPasswordInput,
	exp expectedUpdateUserPasswordResult,
) {
	ctx = session.WithSession(ctx, s.session)

	result, err := s.resolver.Mutation().UpdateUserPassword(ctx, input)
	assert.ErrorIs(t, err, exp.err)
	if err != nil {
		return
	}
	result.User.Scrub()
	assert.Equal(t, exp.result, result)
}

type expectedResendEmailVerificationResult struct {
	err    error
	result bool
}

func (s suite) resendEmailVerification(
	ctx context.Context,
	t *testing.T,
	input model.ResendEmailVerificationInput,
	exp expectedResendEmailVerificationResult,
) {
	ctx = session.WithSession(ctx, s.session)

	result, err := s.resolver.Mutation().ResendEmailVerification(ctx, input)
	assert.ErrorIs(t, err, exp.err)
	if err != nil {
		return
	}
	assert.Equal(t, exp.result, result)
}

type expectedForgotPasswordResult struct {
	err    error
	result bool

	addrExists bool
	addr       string
}

func (s *suite) forgotPassword(
	ctx context.Context,
	t *testing.T,
	input model.ForgotPasswordInput,
	exp expectedForgotPasswordResult,
) {
	result, err := s.resolver.Mutation().ForgotPassword(ctx, input)
	assert.ErrorIs(t, err, exp.err)
	if err != nil {
		return
	}
	assert.Equal(t, exp.result, result)

	if !exp.addrExists {
		return
	}
	email, ok := s.emailer.passwordResets.lpop()
	assert.True(t, ok)
	assert.Equal(t, email.addr, exp.addr)
	s.resetPasswordHash = email.hash
}

type expectedChangePasswordResult struct {
	err    error
	result bool
}

func (s *suite) changePassword(
	ctx context.Context,
	t *testing.T,
	input model.ChangePasswordInput,
	exp expectedChangePasswordResult,
) {
	result, err := s.resolver.Mutation().ChangePassword(ctx, input)
	assert.ErrorIs(t, err, exp.err)
	if err != nil {
		return
	}
	assert.Equal(t, exp.result, result)
}

// --- setup ---

func setup(t *testing.T) *suite {
	// setup data store
	gorm, err := db.Open(config.DSN())
	require.Nil(t, err)

	err = db.Migrate(gorm, config.Migrations())
	require.Nil(t, err)

	store := db.NewStore(zap.NewNop(), gorm)

	// setup redis kv store
	redis := redis.NewClient(&redis.Options{Addr: config.RedisAddr(), Password: config.RedisPassword()})
	sessionManager := session.NewManager(redis)

	// setup emailer
	emailer := &emailer{
		passwordResets: make(emails, 0),
		verifies:       make(emails, 0),
	}

	ctrl := controller.New(sessionManager, store, emailer)
	resolver := graph.NewResolver(
		zap.NewNop(),
		ctrl,
		"localhost",
		false,
	)

	return &suite{
		resolver: resolver,
		emailer:  emailer,
	}
}

// --- mocks ---

type emails []email

func (e emails) lpop() (email, bool) {
	if len(e) <= 0 {
		return email{}, false
	}

	var entry email
	entry, e = e[0], e[1:]

	return entry, true
}

type email struct {
	addr string
	hash string
}

type emailer struct {
	mutex          sync.Mutex
	passwordResets emails
	verifies       emails
}

func (e *emailer) SendPasswordReset(ctx context.Context, addr string, hash string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.passwordResets = append(e.passwordResets, email{addr: addr, hash: hash})
	return nil
}

func (e *emailer) SendVerifyEmail(ctx context.Context, addr string, hash string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.verifies = append(e.verifies, email{addr: addr, hash: hash})
	return nil
}
