package db

import (
	"log"
	"os"
	"time"

	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/internal/migrate"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Open opens a connection with crons' Postgres DB.
func Open(dsn string) (*gorm.DB, error) {
	return gorm.Open(
		postgres.Open(dsn),
		&gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix: "payment.",
			},
			Logger: logger.New(
				log.New(os.Stdout, "\r\n", log.LstdFlags),
				logger.Config{
					SlowThreshold:             200 * time.Millisecond,
					Colorful:                  false,
					IgnoreRecordNotFoundError: true,
					LogLevel:                  logger.Error,
				},
			),
		},
	)
}

func Migrate(db *gorm.DB, migrations string) error {
	dbconn, err := db.DB()
	if err != nil {
		return err
	}
	migration, err := migrate.New(dbconn, migrations)
	if err != nil {
		return err
	}
	if err := migration.Up(); err != nil {
		return err
	}

	return db.AutoMigrate(
		model.Subscription{},
		model.Invoice{},
		model.PaymentIntent{},
	)
}
