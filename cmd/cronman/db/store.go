package db

import (
	"context"

	"github.com/tjper/rustcron/cmd/cronman/model"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IStore interface {
	Tx(func(IStore) error) error

	Create(context.Context, interface{}) error
	Update(context.Context, interface{}, interface{}) error

	ListServers(context.Context, interface{}) error
	ListActiveServerEvents(context.Context) (model.Events, error)

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
func (s Store) ListActiveServerEvents(ctx context.Context) (model.Events, error) {
	events := make(model.Events, 0)
	if res := s.db.
		Model(&model.Event{}).
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
		Preload("Server").
		Preload("Server.Tags").
		Preload("Server.Events").
		Preload("Server.Moderators").
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
			ServerID:      dormant.ServerID,
			AssociationID: input.AssociationID,
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
			ServerID: live.ServerID,
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
			ServerID: dormant.ServerID,
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
