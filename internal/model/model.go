package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Model contains standard attributes that are common among all models.
type Model struct {
	ID uuid.UUID `json:"id" gorm:"default:gen_random_uuid()"`
	At
}

// Scrub removes unpredictable data from the Model instance.
// TODO: rename to Deterministic.
func (m *Model) Scrub() {
	m.ID = uuid.Nil
	m.At.Scrub()
}

// At contains standard time related attributes that are common among all
// models.
type At struct {
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"deletedAt"`
}

// Scrub removes unpredictable data from the Model instance.
// TODO: rename to Deterministic.
func (at *At) Scrub() {
	at.CreatedAt = time.Time{}
	at.UpdatedAt = time.Time{}
	at.DeletedAt = gorm.DeletedAt{}
}
