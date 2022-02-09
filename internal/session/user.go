package session

import (
	"sort"

	"github.com/google/uuid"
)

// User is a session user. It is expected that this type is used across
// applications to represent users via stateful sessions.
type User struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Role  Role      `json:"role"`

	VIPs []uuid.UUID `json:"vips"`
}

func (u User) Equal(u2 User) bool {
	equal := true
	equal = equal && (u.ID == u2.ID)
	equal = equal && (u.Email == u2.Email)
	equal = equal && (u.Role == u2.Role)

	sort.Slice(u.VIPs, func(i, j int) bool { return u.VIPs[i].String() < u.VIPs[j].String() })
	sort.Slice(u2.VIPs, func(i, j int) bool { return u2.VIPs[i].String() < u2.VIPs[j].String() })

	for i := range u.VIPs {
		equal = equal && (u.VIPs[i] == u2.VIPs[i])
	}

	return equal
}

type Role string

const (
	RoleStandard Role = "user"
	RoleAdmin    Role = "admin"
)
