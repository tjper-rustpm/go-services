package migrate

import (
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// New creates a new Migration object.
func New(dbconn *sql.DB, migrations string) (*Migration, error) {
	driver, err := postgres.WithInstance(dbconn, &postgres.Config{})
	if err != nil {
		return nil, err
	}
	migration, err := migrate.NewWithDatabaseInstance(
		migrations,
		"postgres",
		driver,
	)
	if err != nil {
		return nil, err
	}
	return &Migration{
		Migrate: migration,
	}, nil
}

// Migration represents a DB migration.
type Migration struct {
	*migrate.Migrate
}

// Up applies all up migrations to the latest version.
func (m Migration) Up() error {
	err := m.Migrate.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}
