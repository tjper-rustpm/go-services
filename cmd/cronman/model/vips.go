package model

import (
	"context"
	"fmt"
	"time"

	"github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Vip is a "very important person" on a cronman server. The are granted
// special privileges such as queue skip.
type Vip struct {
	model.Model
	ServerID       uuid.UUID
	SubscriptionID uuid.UUID
	SteamID        string
	ExpiresAt      time.Time
}

// Create creates the Vip in the specified db.
func (v *Vip) Create(ctx context.Context, db *gorm.DB) error {
	if err := db.WithContext(ctx).Create(v).Error; err != nil {
		return fmt.Errorf("model Vip.Create: %w", err)
	}
	return nil
}
