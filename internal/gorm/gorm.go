package gorm

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Open opens a connection with the specified DSN.
func Open(dsn string, options ...Option) (*gorm.DB, error) {
	cfg := &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				Colorful:                  false,
				IgnoreRecordNotFoundError: true,
				LogLevel:                  logger.Error,
			},
		),
	}

	for _, option := range options {
		option(cfg)
	}

	return gorm.Open(postgres.Open(dsn), cfg)
}

type Option func(*gorm.Config)

func WithTablePrefix(prefix string) Option {
	return func(c *gorm.Config) {
		c.NamingStrategy = schema.NamingStrategy{TablePrefix: prefix}
	}
}
