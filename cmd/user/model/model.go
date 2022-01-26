package model

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/tjper/rustcron/internal/model"
	"github.com/tjper/rustcron/internal/session"

	"github.com/google/uuid"
)

type User struct {
	model.Model
	Email    string       `json:"email" gorm:"uniqueIndex"`
	Password []byte       `json:"-"`
	Salt     string       `json:"-"`
	Role     session.Role `json:"role"`

	VerificationHash   string       `json:"-" gorm:"uniqueIndex"`
	VerificationSentAt time.Time    `json:"-"`
	VerifiedAt         sql.NullTime `json:"verifiedAt"`

	PasswordResets []PasswordReset `json:"-"`
}

func (u User) MarshalJSON() ([]byte, error) {
	kv := make(map[string]interface{})

	kv["email"] = u.Email
	kv["role"] = u.Role

	if u.VerifiedAt.Valid {
		kv["verifiedAt"] = u.VerifiedAt.Time
	}

	return json.Marshal(kv)
}

func (u User) IsVerified() bool {
	return u.VerifiedAt.Valid
}

func (u User) IsVerificationHashStale() bool {
	return time.Since(u.VerificationSentAt) > 30*time.Minute
}

func (u User) ToSessionUser() session.User {
	return session.User{
		ID:    u.ID,
		Email: u.Email,
		Role:  u.Role,
	}
}

type PasswordReset struct {
	model.Model
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
