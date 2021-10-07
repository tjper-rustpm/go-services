package db

import (
	"context"
	"errors"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/db/model"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IStore interface {
	Tx(func(IStore) error) error

	Create(context.Context, interface{}) error
	Update(context.Context, interface{}, interface{}) error
	Delete(context.Context, interface{}, uuid.UUID) error

	CreateDefinition(context.Context, model.ServerDefinition) (*model.ServerDefinition, error)
	GetDefinition(context.Context, uuid.UUID) (*model.ServerDefinition, error)
	UpdateServerDefinition(context.Context, uuid.UUID, map[string]interface{}) (*model.ServerDefinition, error)
	DefinitionIsLive(context.Context, uuid.UUID) (bool, error)

	ListServers(context.Context, interface{}) error
	ListActiveServerEvents(context.Context) (model.DefinitionEvents, error)
	ListModeratorsPendingRemoval(context.Context, uuid.UUID) (model.DefinitionModerators, error)

	GetLiveServer(context.Context, uuid.UUID) (*model.LiveServer, error)
	GetDormantServer(context.Context, uuid.UUID) (*model.DormantServer, error)

	MakeServerLive(context.Context, MakeServerLiveInput) (*model.LiveServer, error)
	MakeServerDormant(context.Context, uuid.UUID) (*model.DormantServer, error)
	MakeServerArchived(context.Context, uuid.UUID) (*model.ArchivedServer, error)
}

func NewStore(
	logger *zap.Logger,
	db *gorm.DB,
) *Store {
	return &Store{
		logger: logger,
		db:     db,
	}
}

type Store struct {
	logger *zap.Logger
	db     *gorm.DB
}

func (s Store) Tx(fn func(IStore) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return fn(
			Store{logger: s.logger, db: tx},
		)
	})
}

// CreateDefinition creates a new server definition in the store.
func (s Store) CreateDefinition(
	ctx context.Context,
	definition model.ServerDefinition,
) (*model.ServerDefinition, error) {
	if res := s.db.WithContext(ctx).Create(&definition); res.Error != nil {
		return nil, res.Error
	}
	return &definition, nil
}

func (s Store) GetDefinition(
	ctx context.Context,
	id uuid.UUID,
) (*model.ServerDefinition, error) {
	definition := new(model.ServerDefinition)
	if res := s.db.
		WithContext(ctx).
		Preload("Tags").
		Preload("Events").
		Preload("Moderators").
		First(definition, id); res.Error != nil {
		return nil, res.Error
	}
	return definition, nil
}

func (s Store) UpdateServerDefinition(
	ctx context.Context,
	id uuid.UUID,
	changes map[string]interface{},
) (*model.ServerDefinition, error) {

	updates := make(map[string]interface{})
	update := func(changeField, dbField string) {
		val, ok := changes[changeField]
		if !ok {
			return
		}
		updates[dbField] = val
	}
	update("name", "name")
	update("maxPlayers", "max_players")
	update("mapSize", "map_size")
	update("mapSeed", "map_seed")
	update("mapSalt", "map_salt")
	update("tickRate", "tick_rate")
	update("rconPassword", "rcon_password")
	update("description", "description")
	update("url", "url")
	update("background", "background")
	update("bannerURL", "banner_url")
	update("wipeDay", "wipe_day")
	update("blueprintWipeFrequency", "blueprint_wipe_frequency")
	update("mapWipeFrequency", "map_wipe_frequency")

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)

		if res := tx.First(&model.ServerDefinition{}, id); res.Error != nil {
			return res.Error
		}
		if res := tx.Model(
			&model.ServerDefinition{Model: model.Model{ID: id}},
		).Updates(updates); res.Error != nil {
			return res.Error
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return s.GetDefinition(ctx, id)
}

func (s Store) DefinitionIsLive(ctx context.Context, id uuid.UUID) (bool, error) {
	res := s.db.
		WithContext(ctx).
		Where("server_definition_id = ?", id).
		First(&model.LiveServer{})
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if res.Error != nil {
		return false, res.Error
	}
	return true, nil
}

func (s Store) Create(ctx context.Context, value interface{}) error {
	if res := s.db.WithContext(ctx).Create(value); res.Error != nil {
		return res.Error
	}
	return nil
}

func (s Store) Update(ctx context.Context, model interface{}, changes interface{}) error {
	if res := s.db.WithContext(ctx).Model(model).Updates(changes); res.Error != nil {
		return res.Error
	}
	return nil
}

func (s Store) Delete(ctx context.Context, model interface{}, id uuid.UUID) error {
	if res := s.db.WithContext(ctx).Delete(model, id); res.Error != nil {
		return res.Error
	}
	return nil
}

// ListServers
func (s Store) ListServers(ctx context.Context, dst interface{}) error {
	if res := s.db.
		WithContext(ctx).
		Preload("ServerDefinition").
		Preload("ServerDefinition.Tags").
		Preload("ServerDefinition.Events").
		Preload("ServerDefinition.Moderators").
		Order("created_at DESC").
		Find(dst); res.Error != nil {
		return res.Error
	}
	return nil
}

// ListActiveServerEvents
func (s Store) ListActiveServerEvents(ctx context.Context) (model.DefinitionEvents, error) {
	events := make(model.DefinitionEvents, 0)
	if res := s.db.
		Model(&model.DefinitionEvent{}).
		Where(
			"EXISTS (?)",
			s.db.
				Model(&model.LiveServer{}).
				Select("1").
				Where("live_servers.server_definition_id = definition_events.server_definition_id"),
		).
		Or(
			"EXISTS (?)",
			s.db.
				Model(&model.DormantServer{}).
				Select("1").
				Where("dormant_servers.server_definition_id = definition_events.server_definition_id"),
		).
		Find(&events); res.Error != nil {
		return nil, res.Error
	}
	return events, nil
}

// ListModeratorsPendingRemoval
func (s Store) ListModeratorsPendingRemoval(ctx context.Context, id uuid.UUID) (model.DefinitionModerators, error) {
	moderators := make(model.DefinitionModerators, 0)
	if res := s.db.
		Where(
			"server_definition_id = ? AND queued_deletion_at IS NOT NULL",
			id,
			time.Time{},
		).
		Find(&moderators); res.Error != nil {
		return nil, res.Error
	}
	return moderators, nil
}

func (s Store) GetLiveServer(ctx context.Context, id uuid.UUID) (*model.LiveServer, error) {
	server := new(model.LiveServer)
	return server, s.GetServer(ctx, id, server)
}

func (s Store) GetDormantServer(ctx context.Context, id uuid.UUID) (*model.DormantServer, error) {
	server := new(model.DormantServer)
	return server, s.GetServer(ctx, id, server)
}

func (s Store) GetArchivedServer(ctx context.Context, id uuid.UUID) (*model.ArchivedServer, error) {
	server := new(model.ArchivedServer)
	return server, s.GetServer(ctx, id, server)
}

func (s Store) GetServer(ctx context.Context, id uuid.UUID, dst interface{}) error {
	if res := s.db.
		WithContext(ctx).
		Preload("ServerDefinition").
		Preload("ServerDefinition.Tags").
		Preload("ServerDefinition.Events").
		Preload("ServerDefinition.Moderators").
		First(dst, id); res.Error != nil {
		return res.Error
	}
	return nil
}

type MakeServerLiveInput struct {
	ID            uuid.UUID
	AssociationID string
}

func (s Store) MakeServerLive(ctx context.Context, input MakeServerLiveInput) (*model.LiveServer, error) {
	var server *model.LiveServer
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)

		dormant := new(model.DormantServer)
		if res := tx.First(dormant, input.ID); res.Error != nil {
			return res.Error
		}
		if res := tx.Delete(dormant, input.ID); res.Error != nil {
			return res.Error
		}

		server = &model.LiveServer{
			ServerDefinitionID: dormant.ServerDefinitionID,
			AssociationID:      input.AssociationID,
		}
		if res := tx.Create(server); res.Error != nil {
			return res.Error
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s.GetLiveServer(ctx, server.ID)
}

func (s Store) MakeServerDormant(ctx context.Context, id uuid.UUID) (*model.DormantServer, error) {
	var server *model.DormantServer
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)

		live := new(model.LiveServer)
		if res := tx.First(live, id); res.Error != nil {
			return res.Error
		}
		if res := tx.Delete(live, id); res.Error != nil {
			return res.Error
		}

		server = &model.DormantServer{
			ServerDefinitionID: live.ServerDefinitionID,
		}
		if res := tx.Create(server); res.Error != nil {
			return res.Error
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s.GetDormantServer(ctx, server.ID)
}

func (s Store) MakeServerArchived(ctx context.Context, id uuid.UUID) (*model.ArchivedServer, error) {
	var server *model.ArchivedServer
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)

		dormant := new(model.DormantServer)
		if res := tx.First(dormant, id); res.Error != nil {
			return res.Error
		}
		if res := tx.Delete(dormant, id); res.Error != nil {
			return res.Error
		}

		server = &model.ArchivedServer{
			ServerDefinitionID: dormant.ServerDefinitionID,
		}
		if res := tx.Create(server); res.Error != nil {
			return res.Error
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s.GetArchivedServer(ctx, server.ID)
}
