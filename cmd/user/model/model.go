package model

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/tjper/rustcron/internal/model"
	"github.com/tjper/rustcron/internal/session"

	"github.com/google/uuid"
)

type User struct {
	model.Model
	Email    string       `json:"email" gorm:"uniqueIndex,not null"`
	Password []byte       `json:"-" gorm:"not null"`
	Salt     string       `json:"-" gorm:"not null"`
	Role     session.Role `json:"role" gorm:"not null"`

	VerificationHash   string       `json:"-" gorm:"uniqueIndex,not null"`
	VerificationSentAt time.Time    `json:"-" gorm:"not null"`
	VerifiedAt         sql.NullTime `json:"verifiedAt"`

	PasswordResets []PasswordReset `json:"-"`

	Subscriptions []Subscription `json:"subscriptions"`
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

func (u User) IsPassword(password []byte) bool {
	return subtle.ConstantTimeCompare(u.Password, password) == 1
}

func (u User) ToSessionUser() session.User {
	subscriptions := make([]session.Subscription, 0)
	for _, sub := range u.Subscriptions {
		subscriptions = append(
			subscriptions,
			session.Subscription{ID: sub.SubscriptionID, ServerID: sub.ServerID})
	}

	return session.User{
		ID:            u.ID,
		Email:         u.Email,
		Role:          u.Role,
		Subscriptions: subscriptions,
	}
}

type PasswordReset struct {
	model.Model
	ResetHash   string    `gorm:"uniqueIndex,not null"`
	RequestedAt time.Time `gorm:"not null"`
	CompletedAt sql.NullTime

	UserID uuid.UUID `gorm:"not null"`
}

func (r PasswordReset) IsRequestStale() bool {
	return time.Since(r.RequestedAt) > 30*time.Minute
}

func (r PasswordReset) IsCompleted() bool {
	return r.CompletedAt.Valid
}

type Subscription struct {
	model.Model
	UserID         uuid.UUID `gorm:"not null"`
	SubscriptionID uuid.UUID `gorm:"not null"`
	ServerID       uuid.UUID `gorm:"not null"`
}
