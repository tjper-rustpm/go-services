package session

import (
	"time"

	"github.com/google/uuid"
)

func New(
	id string,
	user User,
	absoluteDuration time.Duration,
) *Session {
	return &Session{
		ID:                 id,
		User:               user,
		AbsoluteExpiration: time.Now().Add(absoluteDuration),
		LastActivityAt:     time.Now(),
		CreatedAt:          time.Now(),
	}
}

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

// Equal checks if the passed Session is equal to the receiver Session.
func (s Session) Equal(s2 Session) bool {
	equal := true
	equal = equal && (s.ID == s2.ID)
	equal = equal && s.User.Equal(s2.User)
	equal = equal && s.AbsoluteExpiration.Equal(s2.AbsoluteExpiration)
	equal = equal && s.LastActivityAt.Equal(s2.LastActivityAt)
	equal = equal && s.CreatedAt.Equal(s2.CreatedAt)

	return equal
}
