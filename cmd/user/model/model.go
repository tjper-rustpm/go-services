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
	Email    string              `json:"email" gorm:"uniqueIndex"`
	Password []byte              `json:"-"`
	Salt     string              `json:"-"`
	Role     graphmodel.RoleKind `json:"role"`

	VerificationHash   string       `json:"-" gorm:"uniqueIndex"`
	VerificationSentAt time.Time    `json:"-"`
	VerifiedAt         sql.NullTime `json:"verifiedAt"`

	PasswordResets []PasswordReset `json:"-"`
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

func (r PasswordReset) IsCompleted() bool {
	return r.CompletedAt.Valid
}

type Model struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"deletedAt" gorm:"index"`
}

func (m *Model) Scrub() {
	m.ID = uuid.Nil
	m.CreatedAt = time.Time{}
	m.UpdatedAt = time.Time{}
	m.DeletedAt = gorm.DeletedAt{}
}
