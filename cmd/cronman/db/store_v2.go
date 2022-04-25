package db

import "gorm.io/gorm"

// NewStoreV2 creates a StoreV2 instance.
func NewStoreV2(
	db *gorm.DB,
) *StoreV2 {
	return &StoreV2{db: db}
}

// StoreV2 is responsible for cronman related db interactions. V2 differs
// from Store in that most db logic exists within the model package and StoreV2
// only serves to provide an interface layer for easy mocking.
type StoreV2 struct {
	db *gorm.DB
}
