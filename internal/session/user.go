package session

import (
	"time"

	"github.com/tjper/rustcron/internal/graph/model"

	"github.com/google/uuid"
)

// User is a session user. It is expected that this type is used across
// applications to represent users via stateful sessions.
type User struct {
	ID         uuid.UUID
	Email      string
	Role       model.RoleKind
	VerifiedAt *time.Time
	UpdatedAt  time.Time
	CreatedAt  time.Time
}