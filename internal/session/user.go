package session

import (
	"time"

	"github.com/tjper/rustcron/internal/graph/model"

	"github.com/google/uuid"
)

// User is a session user. It is expected that this type is used across
// applications to represent users via stateful sessions.
type User struct {
	ID         uuid.UUID      `json:"id"`
	Email      string         `json:"email"`
	Role       model.RoleKind `json:"role"`
	VerifiedAt *time.Time     `json:"verifiedAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	CreatedAt  time.Time      `json:"createdAt"`
}
