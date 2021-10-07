package model

import (
	"database/sql"
	"time"

	graphmodel "github.com/tjper/rustcron/internal/graph/model"
	"github.com/tjper/rustcron/internal/session"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	Model
	Email    string `gorm:"uniqueIndex"`
	Password []byte
	Salt     string
	Role     graphmodel.RoleKind

	VerificationHash   string `gorm:"uniqueIndex"`
	VerificationSentAt time.Time
	VerifiedAt         sql.NullTime

	PasswordResets []PasswordReset
}

func (u User) IsVerified() bool {
	return u.VerifiedAt.Valid
}

func (u User) IsVerificationHashStale() bool {
	return time.Since(u.VerificationSentAt) > 30*time.Minute
}

func (u User) ToSessionUser() session.User {
	var verifiedAt *time.Time
	if u.VerifiedAt.Valid {
		verifiedAt = &u.VerifiedAt.Time
	}
	return session.User{
		ID:         u.ID,
		Email:      u.Email,
		Role:       u.Role,
		VerifiedAt: verifiedAt,
		UpdatedAt:  u.UpdatedAt,
		CreatedAt:  u.CreatedAt,
	}
}

type PasswordReset struct {
	Model
	ResetHash   string `gorm:"uniqueIndex"`
	RequestedAt time.Time
	CompletedAt sql.NullTime

	UserID uuid.UUID
}

func (r PasswordReset) IsRequestStale() bool {
	return time.Since(r.RequestedAt) > 30*time.Minute
}

type Model struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (m *Model) Scrub() {
	m.ID = uuid.Nil
	m.CreatedAt = time.Time{}
	m.UpdatedAt = time.Time{}
	m.DeletedAt = gorm.DeletedAt{}
}
