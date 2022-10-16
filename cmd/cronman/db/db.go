package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/model"
	igorm "github.com/tjper/rustcron/internal/gorm"
	"github.com/tjper/rustcron/internal/migrate"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Open opens a connection with crons' Postgres DB.
func Open(dsn string) (*gorm.DB, error) {
	return igorm.Open(dsn, igorm.WithTablePrefix("servers."))
}

// Migrate migrates the gorm.DB utilizing the migrations directory specified.
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

var (
	// ErrServerNotLive indicates an operation was performed assuming a server is
	// live that is not live.
	ErrServerNotLive = errors.New("server is not live")

	// ErrServerDNE indicates an operation was performed on a server that does
	// not exist.
	ErrServerDNE = errors.New("server does not exist")
)

// UpdateLiveServerInfo encompasses all logic to update the server info of a
// live server.
type UpdateLiveServerInfo struct {
	// LiveServerID is the unique identifier of the live server to be updated.
	LiveServerID uuid.UUID
	// Changes are the field and value pairs that are to be updated.
	Changes map[string]interface{}
}

// Exec implements the igorm.Execer interface.
func (u UpdateLiveServerInfo) Exec(ctx context.Context, db *gorm.DB) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var server model.LiveServer
		if err := tx.First(&server, u.LiveServerID).Error; err != nil {
			return err
		}

		return tx.Model(&server).Updates(u.Changes).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrServerNotLive
	}
	if err != nil {
		return fmt.Errorf("while updating live server info: %w", err)
	}
	return nil
}

// CreateWipe encompasses all logic to create a new server wipe.
type CreateWipe struct {
	ServerID uuid.UUID
	Wipe     model.Wipe
}

// Exec implements the igorm.Execer interface.
func (c CreateWipe) Exec(ctx context.Context, db *gorm.DB) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var server model.Server
		if err := tx.First(&server, c.ServerID).Error; err != nil {
			return err
		}

		c.Wipe.ServerID = server.ID
		return tx.Create(&c.Wipe).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrServerDNE
	}
	if err != nil {
		return fmt.Errorf("while creating server wipe: %w", err)
	}

	return nil
}

// UpdateWipeApplied encompasses all logic to update a wipe to indicate it has
// been applied.
type UpdateWipeApplied struct {
	WipeID uuid.UUID
}

// Exec implements the igorm.Execer interface.
func (u UpdateWipeApplied) Exec(ctx context.Context, db *gorm.DB) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var wipe model.Wipe
		if err := tx.First(&wipe, u.WipeID).Error; err != nil {
			return err
		}

		return tx.Model(&wipe).Update("applied_at", time.Now()).Error
	})
	if err != nil {
		return fmt.Errorf("while updating wipe to applied: %w", err)
	}
	return nil
}

// FindDormantServer encompasses all logic to retrieve a dormant server.
type FindDormantServer struct {
	ServerID uuid.UUID
	Result   model.DormantServer
}

// Find implements the igorm.Finder interface.
func (f *FindDormantServer) Find(ctx context.Context, db *gorm.DB) error {
	var server model.Server
	err := db.
		WithContext(ctx).
		Model(&server).
		Preload("Wipes").
		Preload("Tags").
		Preload("Events").
		Preload("Moderators").
		Preload("Vips").
		First(&server, f.ServerID).Error
	if err != nil {
		return fmt.Errorf("while retrieving server: %w", err)
	}

	var dormant model.DormantServer
	err = db.
		WithContext(ctx).
		Model(&dormant).
		First(&dormant, server.StateID).Error
	if err != nil {
		return fmt.Errorf("while retrieving dormant server: %w", err)
	}

	dormant.Server = server
	f.Result = dormant

	return nil
}
