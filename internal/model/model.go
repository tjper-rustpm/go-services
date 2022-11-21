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

// Equal checks if the Model instance is equal to the passed Model instance.
func (m Model) Equal(m2 Model) bool {
	equal := true
	equal = equal && m.ID == m2.ID
	equal = equal && m.At.Equal(m2.At)
	return equal
}

// Scrub removes unpredictable data from the Model instance.
func (m *Model) Scrub() {
	m.ID = uuid.Nil
	m.At.Scrub()
}

// At contains standard time related attributes that are common among all
// models.
type At struct {
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `json:"-"`
}

// Equal checks if the At instance is equal to the passed At instance.
func (at At) Equal(at2 At) bool {
	equal := true
	equal = equal && at.CreatedAt.Equal(at2.CreatedAt)
	equal = equal && at.UpdatedAt.Equal(at2.UpdatedAt)
	equal = equal && at.DeletedAt.Time.Equal(at2.DeletedAt.Time)
	return equal
}

// Scrub removes unpredictable data from the Model instance.
func (at *At) Scrub() {
	at.CreatedAt = time.Time{}
	at.UpdatedAt = time.Time{}
	at.DeletedAt = gorm.DeletedAt{}
}
