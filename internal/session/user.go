package session

import (
	"github.com/google/uuid"
)

// User is a session user. It is expected that this type is used across
// applications to represent users via stateful sessions.
type User struct {
	ID      uuid.UUID `json:"id"`
	Email   string    `json:"email"`
	Role    Role      `json:"role"`
	SteamID string    `json:"steamId"`
}

func (u User) Equal(u2 User) bool {
	equal := true
	equal = equal && (u.ID == u2.ID)
	equal = equal && (u.Email == u2.Email)
	equal = equal && (u.Role == u2.Role)
	equal = equal && (u.SteamID == u2.SteamID)

	return equal
}

func (u User) IsSteamIDAssociated() bool {
	return u.SteamID != ""
}

type Role string

const (
	RoleStandard Role = "user"
	RoleAdmin    Role = "admin"
)
