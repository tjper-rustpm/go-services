package session

import (
	"time"

	"github.com/google/uuid"
)

// Session represents a client Session.
type Session struct {
	// ID is the unique identifier of the Session. This identifier needs to be
	// crytographically secure pseudo-random number.
	ID string `json:"id"`

	// User is the session User.
	User User `json:"user"`

	// AbsoluteExpiration is the time at which the Session is considered expired
	// regardless of recent activity. User must then re-authenticate with
	// service.
	AbsoluteExpiration time.Time `json:"absoluteExpiration"`

	// LastActivityAt is the last time the Session was interacted with.
	LastActivityAt time.Time `json:"lastActivityAt"`

	// CreatedAt is the time the Session was created.
	CreatedAt time.Time `json:"createdAt"`
}

// IsAuthorized ensures that the session is authorized to interact with the
// specified user ID.
func (s Session) IsAuthorized(userID uuid.UUID) bool {
	return s.User.ID == userID
}

// Equal checks if the passed Session is equal to the reciever Session.
func (s Session) Equal(s2 Session) bool {
	if s.ID != s2.ID {
		return false
	}
	if !s.LastActivityAt.Equal(s2.LastActivityAt) {
		return false
	}
	return true
}
