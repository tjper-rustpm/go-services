package model

import (
	"time"

	"github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
)

// Vip is a "very important person" on a cronman server. The are granted
// special privileges such as queue skip.
type Vip struct {
	model.Model
	SubscriptionID uuid.UUID
	UserID         uuid.UUID
	SteamID        string
	ExpiresAt      time.Time
}
