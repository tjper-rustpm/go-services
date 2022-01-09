package db

import (
	"context"
	"errors"
	"fmt"

	cronmanerrors "github.com/tjper/rustcron/cmd/cronman/errors"
	"github.com/tjper/rustcron/cmd/cronman/model"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IStore interface {
	Tx(func(IStore) error) error

	CreateServer(context.Context, model.Server) (*model.DormantServer, error)

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

func (s Store) CreateServer(ctx context.Context, srv model.Server) (*model.DormantServer, error) {
	var dormant model.DormantServer

	dormant.Server = srv
	if res := s.db.Create(&dormant); res.Error != nil {
		return nil, res.Error
	}

	return &dormant, nil
}

func (s Store) ListServers(ctx context.Context, dst interface{}) error {
	if res := s.db.
		WithContext(ctx).
		Preload("Server").
		Preload("Server.Tags").
		Preload("Server.Events").
		Preload("Server.Moderators").
		Order("created_at DESC").
		Find(dst); res.Error != nil {
		return res.Error
	}

	return nil
}

func (s Store) ListActiveServerEvents(ctx context.Context) (model.Events, error) {
	events := make(model.Events, 0)
	if res := s.db.
		Model(&model.Event{}).
		Where(
			"EXISTS (?)",
			s.db.
				Model(&model.LiveServer{}).
				Select("1").
				Where("live_servers.server_id = events.server_id"),
		).
		Or(
			"EXISTS (?)",
			s.db.
				Model(&model.DormantServer{}).
				Select("1").
				Where("dormant_servers.server_id = events.server_id"),
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
	res := s.db.
		WithContext(ctx).
		Preload("Server").
		Preload("Server.Tags").
		Preload("Server.Events").
		Preload("Server.Moderators").
		First(dst, id)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("get server; id: %s, error: %w", id, cronmanerrors.ErrServerDNE)
	}
	if res.Error != nil {
		return fmt.Errorf("get server; id: %s, error: %w", id, res.Error)
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

		var dormant model.DormantServer
		if res := tx.Preload("Server").First(&dormant, input.ID); res.Error != nil {
			return fmt.Errorf("select dormant server; %w", res.Error)
		}

		server = &model.LiveServer{
			Server:        dormant.Server,
			AssociationID: input.AssociationID,
		}
		if res := tx.Create(server); res.Error != nil {
			return res.Error
		}

		if res := tx.Delete(&dormant); res.Error != nil {
			return fmt.Errorf("delete dormant server; %w", res.Error)
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

		var live model.LiveServer
		if res := tx.Preload("Server").First(&live, id); res.Error != nil {
			return fmt.Errorf("select live server; %w", res.Error)
		}

		server = &model.DormantServer{
			Server: live.Server,
		}
		if res := tx.Create(server); res.Error != nil {
			return res.Error
		}

		if res := tx.Delete(&live); res.Error != nil {
			return fmt.Errorf("delete live server; %w", res.Error)
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

		var dormant model.DormantServer
		if res := tx.Preload("Server").First(&dormant, id); res.Error != nil {
			return fmt.Errorf("select dormant server; %w", res.Error)
		}

		server = &model.ArchivedServer{
			Server: dormant.Server,
		}
		if res := tx.Create(server); res.Error != nil {
			return res.Error
		}

		if res := tx.Delete(&dormant); res.Error != nil {
			return fmt.Errorf("delete dormant server; %w", res.Error)
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return s.GetArchivedServer(ctx, server.ID)
}
