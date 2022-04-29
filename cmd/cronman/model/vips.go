package model

import (
	"context"
	"fmt"
	"time"

	"github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Vips is a slice of Vip instances.
type Vips []Vip

// FindByServerID retrieves the Vips related to the specified serverID.
func (vs *Vips) FindByServerID(ctx context.Context, db *gorm.DB, serverID uuid.UUID) error {
	if err := db.WithContext(ctx).Where("server_id = ?", serverID).Find(vs).Error; err != nil {
		return fmt.Errorf("model Vips.FindByServerID: %w", err)
	}
	return nil
}

// Active filters and returns retrieves the subset of active Vips.
func (vs Vips) Active() Vips {
	var vips Vips
	for _, vip := range vs {
		if !time.Now().Before(vip.ExpiresAt) {
			continue
		}
		vips = append(vips, vip)
	}
	return vips
}

// SteamIDs retrieves the Vips set of steam IDs.
func (vs Vips) SteamIDs() []string {
	var steamIDs []string
	for _, vip := range vs {
		steamIDs = append(steamIDs, vip.SteamID)
	}
	return steamIDs
}

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
