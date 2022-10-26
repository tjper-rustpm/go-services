package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/iancoleman/strcase"
	cronmanerrors "github.com/tjper/rustcron/cmd/cronman/errors"
	"github.com/tjper/rustcron/cmd/cronman/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func UpdateServer(
	ctx context.Context,
	db *gorm.DB,
	id uuid.UUID,
	changes map[string]interface{},
) (*model.DormantServer, error) {
	current, err := GetDormantServer(ctx, db, id)
	if err != nil {
		return nil, err
	}

	snakeCaseChanges := make(map[string]interface{})
	for field, value := range changes {
		snakeCaseChanges[strcase.ToSnake(field)] = value
	}

	if res := db.
		WithContext(ctx).
		Model(&current.Server).
		Updates(snakeCaseChanges); res.Error != nil {
		return nil, fmt.Errorf("update server; id: %s, error: %w", id, res.Error)
	}

	return GetDormantServer(ctx, db, id)
}

func WipeServer(ctx context.Context, db *gorm.DB, serverID uuid.UUID, wipe model.Wipe) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var server model.Server
		if err := tx.First(&server, serverID).Error; err != nil {
			return err
		}

		wipe.ServerID = server.ID
		return tx.Create(&wipe).Error
	})
}

func ListServers(ctx context.Context, db *gorm.DB, dst interface{}) error {
	if res := db.
		WithContext(ctx).
		Preload("Server").
		Preload("Server.Wipes").
		Preload("Server.Tags").
		Preload("Server.Events").
		Preload("Server.Moderators").
		Order("created_at DESC").
		Find(dst); res.Error != nil {
		return res.Error
	}

	return nil
}

func ListActiveServerEvents(ctx context.Context, db *gorm.DB) (model.Events, error) {
	events := make(model.Events, 0)
	if res := db.
		Model(&model.Event{}).
		Where(
			"EXISTS (?)",
			db.
				Model(&model.Server{}).
				Select("1").
				Where("servers.id = events.server_id").
				Where(
					db.Where("servers.state_type = ?", model.LiveServerState).
						Or("servers.state_type = ?", model.DormantServerState),
				),
		).
		Find(&events); res.Error != nil {
		return nil, res.Error
	}
	return events, nil
}

func ListVipsByServerID(ctx context.Context, db *gorm.DB, serverID uuid.UUID) (model.Vips, error) {
	var vips model.Vips
	if err := db.WithContext(ctx).Where("server_id = ?", serverID).Find(&vips).Error; err != nil {
		return nil, fmt.Errorf("while finding vips by server ID: %w", err)
	}
	return vips, nil
}

func GetLiveServer(ctx context.Context, db *gorm.DB, id uuid.UUID) (*model.LiveServer, error) {
	server, err := GetServer(ctx, db, id)
	if err != nil {
		return nil, fmt.Errorf("get live server: %s, error: %w", id, err)
	}

	var live model.LiveServer
	res := db.First(&live, server.StateID)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf(
			"get live state: %s, error: %w",
			server.StateID,
			cronmanerrors.ErrServerNotLive,
		)
	}
	if res.Error != nil {
		return nil, fmt.Errorf("get live state: %s, error: %w", server.StateID, res.Error)
	}

	live.Server = *server

	return &live, nil
}

func GetDormantServer(ctx context.Context, db *gorm.DB, id uuid.UUID) (*model.DormantServer, error) {
	server, err := GetServer(ctx, db, id)
	if err != nil {
		return nil, fmt.Errorf("get dormant server: %s, error: %w", id, err)
	}

	var dormant model.DormantServer
	res := db.First(&dormant, server.StateID)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf(
			"get dormant state: %s, error: %w",
			server.StateID,
			cronmanerrors.ErrServerNotDormant,
		)
	}
	if res.Error != nil {
		return nil, fmt.Errorf("get dormant state: %s, error: %w", server.StateID, res.Error)
	}

	dormant.Server = *server

	return &dormant, nil
}

func GetArchivedServer(ctx context.Context, db *gorm.DB, id uuid.UUID) (*model.ArchivedServer, error) {
	server, err := GetServer(ctx, db, id)
	if err != nil {
		return nil, fmt.Errorf("get archived server: %s, error: %w", id, err)
	}

	var archived model.ArchivedServer
	res := db.First(&archived, server.StateID)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf(
			"get archived state: %s, error: %w",
			server.StateID,
			cronmanerrors.ErrServerNotArchived,
		)
	}
	if res.Error != nil {
		return nil, fmt.Errorf("get archived state: %s, error: %w", server.StateID, res.Error)
	}

	archived.Server = *server

	return &archived, nil
}

func GetServer(ctx context.Context, db *gorm.DB, id uuid.UUID) (*model.Server, error) {
	var server model.Server
	res := db.
		WithContext(ctx).
		Preload("Wipes").
		Preload("Tags").
		Preload("Events").
		Preload("Owners").
		Preload("Moderators").
		Preload("Vips").
		First(&server, id)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get server; id: %s, error: %w", id, cronmanerrors.ErrServerDNE)
	}
	if res.Error != nil {
		return nil, fmt.Errorf("get server; id: %s, error: %w", id, res.Error)
	}

	return &server, nil
}

type MakeServerLiveInput struct {
	ID            uuid.UUID
	AssociationID string
}

func MakeServerLive(ctx context.Context, db *gorm.DB, input MakeServerLiveInput) (*model.LiveServer, error) {
	dormant, err := GetDormantServer(ctx, db, input.ID)
	if err != nil {
		return nil, err
	}

	var server *model.LiveServer
	if err := db.Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)

		if res := tx.Delete(&dormant); res.Error != nil {
			return fmt.Errorf("delete dormant server; id: %s, error: %w", input.ID, res.Error)
		}

		server = &model.LiveServer{
			Server:        dormant.Server,
			AssociationID: input.AssociationID,
		}
		if res := tx.Create(server); res.Error != nil {
			return fmt.Errorf("create live server; id: %s, error: %w", input.ID, res.Error)
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return server, nil
}

func MakeServerDormant(ctx context.Context, db *gorm.DB, id uuid.UUID) (*model.DormantServer, error) {
	live, err := GetLiveServer(ctx, db, id)
	if err != nil {
		return nil, err
	}

	var server *model.DormantServer
	if err := db.Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)

		if res := tx.Delete(&live); res.Error != nil {
			return fmt.Errorf("delete live server; id: %s, error: %w", id, res.Error)
		}

		server = &model.DormantServer{
			Server: live.Server,
		}
		if res := tx.Create(server); res.Error != nil {
			return fmt.Errorf("create dormant server; id: %s, error: %w", id, res.Error)
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return server, nil
}

func MakeServerArchived(ctx context.Context, db *gorm.DB, id uuid.UUID) (*model.ArchivedServer, error) {
	dormant, err := GetDormantServer(ctx, db, id)
	if err != nil {
		return nil, err
	}

	var server *model.ArchivedServer
	if err := db.Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)

		if res := tx.Delete(&dormant); res.Error != nil {
			return fmt.Errorf("delete dormant server; id: %s, error: %w", id, res.Error)
		}

		server = &model.ArchivedServer{
			Server: dormant.Server,
		}
		if res := tx.Create(server); res.Error != nil {
			return fmt.Errorf("create archived server; id: %s, error: %w", id, res.Error)
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return GetArchivedServer(ctx, db, id)
}

func ApplyWipe(ctx context.Context, db *gorm.DB, wipeID uuid.UUID) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var wipe model.Wipe
		if err := tx.First(&wipe, wipeID).Error; err != nil {
			return err
		}

		return tx.Model(&wipe).Update("applied_at", time.Now()).Error
	})
}

func UpdateLiveServer(
	ctx context.Context,
	db *gorm.DB,
	serverID uuid.UUID,
	changes map[string]interface{},
) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var server model.LiveServer
		if err := tx.First(&server, serverID).Error; err != nil {
			return err
		}

		return tx.Model(&server).Updates(changes).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrServerNotLive
	}
	if err != nil {
		return fmt.Errorf("while updating live server: %w", err)
	}
	return nil
}
