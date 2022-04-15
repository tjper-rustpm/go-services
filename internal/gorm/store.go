package gorm

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// ErrAlreadyExists indicates that an attempt was made to create an entity that
// already exists.
var ErrAlreadyExists = errors.New("entity already exists")

// ErrNotFound indicates the entity was not found.
var ErrNotFound = gorm.ErrRecordNotFound

// Creator encompasses creating an entity in the passed *gorm.DB.
type Creator interface {
	Create(context.Context, *gorm.DB) error
}

// Firster encompasses fetching the entity form the passed *gorm.DB.
type Firster interface {
	First(context.Context, *gorm.DB) error
}

// NewStore creates a new Store instance.
func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// Store provides a mockable API for interacting with a Postgres DB with GORM.
type Store struct {
	db *gorm.DB
}

// Create creates the entity as it has defined.
func (s Store) Create(ctx context.Context, entity Creator) error {
	return entity.Create(ctx, s.db)
}

// First retrieves a single instance of the entity.
func (s Store) First(ctx context.Context, entity Firster) error {
	return entity.First(ctx, s.db)
}
