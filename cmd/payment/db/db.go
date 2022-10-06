package db

import (
	igorm "github.com/tjper/rustcron/internal/gorm"
	"github.com/tjper/rustcron/internal/migrate"

	"gorm.io/gorm"
)

// Open opens a connection with the payment Postgres DB.
func Open(dsn string) (*gorm.DB, error) {
	return igorm.Open(dsn, igorm.WithTablePrefix("payments."))
}

// Migrate migrates the db as the migrations specify.
func Migrate(db *gorm.DB, migrations string) error {
	dbconn, err := db.DB()
	if err != nil {
		return err
	}

	return migrate.Migrate(
		dbconn,
		migrations,
		migrate.WithMigrationsTable("payments_migrations"),
	)
}
