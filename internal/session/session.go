package session

import (
	"time"
)

// Session represents a client Session.
type Session struct {
	// ID is the unique identifier of the Session. This identifier needs to be
	// crytographically secure pseudo-random number.
	ID string

	// LastActivityAt is the last time the Session was interacted with.
	LastActivityAt time.Time

	// User is the session User.
	User User
}

// IsAuthorized ensures that the session is authorized to interact with the
// specified user ID.
func (s Session) IsAuthorized(userID string) bool {
	return s.User.ID.String() == userID
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
