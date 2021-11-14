package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Model struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (m *Model) Scrub() {
	m.ID = uuid.Nil
	m.CreatedAt = time.Time{}
	m.UpdatedAt = time.Time{}
	m.DeletedAt = gorm.DeletedAt{}
}