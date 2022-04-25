package gorm

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrAlreadyExists indicates that an attempt was made to create an entity that
// already exists.
var ErrAlreadyExists = errors.New("entity already exists")

// ErrNotFound indicates the entity was not found.
var ErrNotFound = gorm.ErrRecordNotFound

// NewStore creates a new Store instance.
func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// Store provides a mockable API for interacting with a Postgres DB with GORM.
type Store struct {
	db *gorm.DB
}

// Creator encompasses creating an entity in the passed *gorm.DB.
type Creator interface {
	Create(context.Context, *gorm.DB) error
}

// Create wraps execution of entity.Create.
func (s Store) Create(ctx context.Context, entity Creator) error {
	return entity.Create(ctx, s.db)
}

// Firster encompasses fetching the entity form the passed *gorm.DB. An empty
// result should return an error.
type Firster interface {
	First(context.Context, *gorm.DB) error
}

// First wraps execution of entity.First.
func (s Store) First(ctx context.Context, entity Firster) error {
	return entity.First(ctx, s.db)
}

// Finder encompasses fetching an entity from the passed *gorm.DB. An empty
// result should not return an error.
type Finder interface {
	Find(context.Context, *gorm.DB) error
}

// Find wraps execution of entity.Find.
func (s Store) Find(ctx context.Context, entity Finder) error {
	return entity.Find(ctx, s.db)
}

// FinderByUserID encompasses a type that is able to retrieve itself from
// *gorm.DB by its user ID.
type FinderByUserID interface {
	FindByUserID(context.Context, *gorm.DB, uuid.UUID) error
}

// FindByUserID wraps execution of entity.FindUserByID.
func (s Store) FindByUserID(ctx context.Context, entity FinderByUserID, userID uuid.UUID) error {
	return entity.FindByUserID(ctx, s.db, userID)
}

// FinderByServerID encompasses a type that is able to retrieve itself from
// *gorm.DB by server ID.
type FinderByServerID interface {
	FindByServerID(context.Context, *gorm.DB, uuid.UUID) error
}

// FindByServerID wraps execution of entity.FindByServerID.
func (s Store) FindByServerID(ctx context.Context, entity FinderByServerID, serverID uuid.UUID) error {
	return entity.FindByServerID(ctx, s.db, serverID)
}
