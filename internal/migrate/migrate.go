package migrate

import (
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrate migrates the DB with the specified migrations files.
func Migrate(dbconn *sql.DB, migrations string, options ...Option) error {
	cfg := &postgres.Config{
		MigrationsTable: "migrations",
	}
	for _, option := range options {
		option(cfg)
	}

	driver, err := postgres.WithInstance(dbconn, cfg)
	if err != nil {
		return err
	}

	migration, err := migrate.NewWithDatabaseInstance(migrations, "postgres", driver)
	if err != nil {
		return err
	}

	if err := migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

type Option func(*postgres.Config)

func WithMigrationsTable(name string) Option {
	return func(c *postgres.Config) {
		c.MigrationsTable = name
	}
}
