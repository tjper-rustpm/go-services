package db

import (
	igorm "github.com/tjper/rustcron/internal/gorm"
	"github.com/tjper/rustcron/internal/migrate"

	"gorm.io/gorm"
)

// Open opens a connection with crons' Postgres DB.
func Open(dsn string) (*gorm.DB, error) {
	return igorm.Open(dsn, igorm.WithTablePrefix("servers."))
}

func Migrate(db *gorm.DB, migrations string) error {
	dbconn, err := db.DB()
	if err != nil {
		return err
	}
	return migrate.Migrate(
		dbconn,
		migrations,
		migrate.WithMigrationsTable("servers_migrations"),
	)
}
