package session

import (
	"github.com/google/uuid"
)

// User is a session user. It is expected that this type is used across
// applications to represent users via stateful sessions.
type User struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Role  Role      `json:"role"`

	CustomerID    string         `json:"-"`
	Subscriptions []Subscription `json:"-"`
}

func (u User) Equal(u2 User) bool {
	equal := true
	equal = equal && (u.ID == u2.ID)
	equal = equal && (u.Email == u2.Email)
	equal = equal && (u.Role == u2.Role)
	equal = equal && (u.CustomerID == u2.CustomerID)

	for i := range u.Subscriptions {
		equal = equal && (u.Subscriptions[i] == u2.Subscriptions[i])
	}

	return equal
}

func (u User) SubscriptionIDs() []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(u.Subscriptions))
	for _, sub := range u.Subscriptions {
		ids = append(ids, sub.ID)
	}
	return ids
}

type Role string

const (
	RoleStandard Role = "user"
	RoleAdmin    Role = "admin"
)

type Subscription struct {
	ID       uuid.UUID
	ServerID uuid.UUID
}
