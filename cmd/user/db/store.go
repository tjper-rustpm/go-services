package db

import (
	"context"
	"errors"
	"time"

	rpmerrors "github.com/tjper/rustcron/cmd/user/errors"
	"github.com/tjper/rustcron/cmd/user/model"
	imodel "github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func NewStore(
	logger *zap.Logger,
	db *gorm.DB,
) *Store {
	return &Store{
		logger: logger,
		db:     db,
	}
}

type Store struct {
	logger *zap.Logger
	db     *gorm.DB
}

func (s Store) CreateUser(ctx context.Context, user *model.User) error {
	if res := s.db.WithContext(ctx).Create(user); res.Error != nil {
		return res.Error
	}
	return nil
}

func (s Store) UpdateUserPassword(
	ctx context.Context,
	id uuid.UUID,
	password []byte,
) (*model.User, error) {
	if res := s.db.WithContext(ctx).Model(
		&model.User{Model: imodel.Model{ID: id}},
	).Update("password", password); res.Error != nil {
		return nil, res.Error
	}
	return s.User(ctx, id)
}

func (s Store) User(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user := new(model.User)
	res := s.db.WithContext(ctx).First(user, id)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, rpmerrors.UserDNE
	}
	if res.Error != nil {
		return nil, res.Error
	}
	return user, nil
}

func (s Store) UserByEmail(ctx context.Context, email string) (*model.User, error) {
	user := new(model.User)
	res := s.db.WithContext(ctx).Where("email = ?", email).First(user)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, rpmerrors.UserDNE
	}
	if res.Error != nil {
		return nil, res.Error
	}
	return user, nil
}

func (s Store) UserByResetPasswordHash(ctx context.Context, hash string) (*model.User, error) {
	user := new(model.User)
	res := s.db.
		WithContext(ctx).
		Model(&model.User{}).
		Where(
			"EXISTS (?)",
			s.db.
				Model(&model.PasswordReset{}).
				Select("1").
				Where("password_resets.user_id = users.id").
				Where("password_resets.reset_hash = ?", hash),
		).
		First(user)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, rpmerrors.UserDNE
	}
	if res.Error != nil {
		return nil, res.Error
	}
	return user, nil
}

func (s Store) UserByVerificationHash(ctx context.Context, hash string) (*model.User, error) {
	user := new(model.User)
	res := s.db.WithContext(ctx).First(user, "verification_hash = ?", hash)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, rpmerrors.UserDNE
	}
	if res.Error != nil {
		return nil, res.Error
	}
	return user, nil
}

func (s Store) VerifyEmail(
	ctx context.Context,
	id uuid.UUID,
	hash string,
) (*model.User, error) {
	if res := s.db.WithContext(ctx).Model(
		&model.User{Model: imodel.Model{ID: id}},
	).Update("verified_at", time.Now()); res.Error != nil {
		return nil, res.Error
	}
	return s.User(ctx, id)
}

func (s Store) ResetEmailVerification(
	ctx context.Context,
	id uuid.UUID,
	hash string,
) (*model.User, error) {
	if res := s.db.WithContext(ctx).Model(
		&model.User{Model: imodel.Model{ID: id}},
	).Updates(
		map[string]interface{}{
			"verification_hash":    hash,
			"verification_sent_at": time.Now(),
		},
	); res.Error != nil {
		return nil, res.Error
	}
	return s.User(ctx, id)
}

func (s Store) CreateUserPasswordReset(
	ctx context.Context,
	email string,
	hash string,
) (*model.PasswordReset, error) {
	user, err := s.UserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	reset := &model.PasswordReset{
		UserID:      user.ID,
		ResetHash:   hash,
		RequestedAt: time.Now(),
	}
	if res := s.db.WithContext(ctx).Create(reset); res.Error != nil {
		return nil, res.Error
	}
	return reset, nil
}

func (s Store) PasswordResetByHash(ctx context.Context, hash string) (*model.PasswordReset, error) {
	reset := new(model.PasswordReset)
	res := s.db.WithContext(ctx).First(
		reset,
		"reset_hash = ?",
		hash,
	)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, rpmerrors.PasswordResetDNE
	}
	if res.Error != nil {
		return nil, res.Error
	}
	return reset, nil
}

func (s Store) CompleteUserPasswordReset(
	ctx context.Context,
	userID uuid.UUID,
	passwordResetID uuid.UUID,
	password []byte,
) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if res := tx.
			Model(&model.User{Model: imodel.Model{ID: userID}}).
			Update("password", password); res.Error != nil {
			return res.Error
		}

		if res := tx.
			Model(&model.PasswordReset{Model: imodel.Model{ID: passwordResetID}}).
			Update("completed_at", time.Now()); res.Error != nil {
			return res.Error
		}
		return nil
	})
}
