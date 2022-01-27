package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

type Store struct {
	db *gorm.DB
}

func (s Store) Create(ctx context.Context, obj interface{}) error {
	if res := s.db.WithContext(ctx).Create(obj); res.Error != nil {
		return fmt.Errorf("create; type: %T, error: %w", obj, res.Error)
	}
	return nil
}
