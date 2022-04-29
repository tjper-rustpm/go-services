package db

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"

	"github.com/iancoleman/strcase"
	cronmanerrors "github.com/tjper/rustcron/cmd/cronman/errors"
	"github.com/tjper/rustcron/cmd/cronman/model"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IStore interface {
	Tx(func(IStore) error) error

	Find(context.Context, interface{}, []uuid.UUID) error
	Create(context.Context, interface{}) error
	Delete(context.Context, interface{}, []uuid.UUID) error

	UpdateServer(context.Context, uuid.UUID, map[string]interface{}) (*model.DormantServer, error)
	WipeServer(context.Context, *model.DormantServer) error

	ListServers(context.Context, interface{}) error
	ListActiveServerEvents(context.Context) (model.Events, error)

	GetServer(context.Context, uuid.UUID) (*model.Server, error)
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

func (s Store) Find(ctx context.Context, obj interface{}, ids []uuid.UUID) error {
	if res := s.db.WithContext(ctx).Find(obj, ids); res.Error != nil {
		return fmt.Errorf("find; type: %T, error: %w", obj, res.Error)
	}
	return nil
}

func (s Store) Create(ctx context.Context, obj interface{}) error {
	if res := s.db.Create(obj); res.Error != nil {
		return fmt.Errorf("create; type: %T, error: %w", obj, res.Error)
	}
	return nil
}

func (s Store) Delete(ctx context.Context, obj interface{}, ids []uuid.UUID) error {
	if res := s.db.WithContext(ctx).Delete(obj, ids); res.Error != nil {
		return fmt.Errorf("delete; type: %T, error: %w", obj, res.Error)
	}
	return nil
}

func (s Store) UpdateServer(
	ctx context.Context,
	id uuid.UUID,
	changes map[string]interface{},
) (*model.DormantServer, error) {
	current, err := s.GetDormantServer(ctx, id)
	if err != nil {
		return nil, err
	}

	snakeCaseChanges := make(map[string]interface{})
	for field, value := range changes {
		snakeCaseChanges[strcase.ToSnake(field)] = value
	}

	if res := s.db.
		WithContext(ctx).
		Model(&current.Server).
		Updates(snakeCaseChanges); res.Error != nil {
		return nil, fmt.Errorf("update server; id: %s, error: %w", id, res.Error)
	}

	return s.GetDormantServer(ctx, id)
}

func (s Store) WipeServer(
	ctx context.Context,
	dormant *model.DormantServer,
) error {
	wipe := &model.Wipe{
		MapSeed: uint16(rand.Intn(math.MaxUint16)),
		MapSalt: uint16(rand.Intn(math.MaxUint16)),
	}
	if res := s.db.WithContext(ctx).Create(wipe); res.Error != nil {
		return fmt.Errorf(
			"create wipe; id: %s, error: %w",
			dormant.Server.ID,
			res.Error,
		)
	}

	return nil
}

func (s Store) ListServers(ctx context.Context, dst interface{}) error {
	if res := s.db.
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
	server, err := s.GetServer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get live server: %s, error: %w", id, err)
	}

	var live model.LiveServer
	res := s.db.First(&live, server.StateID)
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

func (s Store) GetDormantServer(ctx context.Context, id uuid.UUID) (*model.DormantServer, error) {
	server, err := s.GetServer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get dormant server: %s, error: %w", id, err)
	}

	var dormant model.DormantServer
	res := s.db.First(&dormant, server.StateID)
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

func (s Store) GetArchivedServer(ctx context.Context, id uuid.UUID) (*model.ArchivedServer, error) {
	server, err := s.GetServer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get archived server: %s, error: %w", id, err)
	}

	var archived model.ArchivedServer
	res := s.db.First(&archived, server.StateID)
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

func (s Store) GetServer(ctx context.Context, id uuid.UUID) (*model.Server, error) {
	var server model.Server
	res := s.db.
		WithContext(ctx).
		Preload("Wipes").
		Preload("Tags").
		Preload("Events").
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

func (s Store) MakeServerLive(ctx context.Context, input MakeServerLiveInput) (*model.LiveServer, error) {
	dormant, err := s.GetDormantServer(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	var server *model.LiveServer
	if err := s.db.Transaction(func(tx *gorm.DB) error {
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

func (s Store) MakeServerDormant(ctx context.Context, id uuid.UUID) (*model.DormantServer, error) {
	live, err := s.GetLiveServer(ctx, id)
	if err != nil {
		return nil, err
	}

	var server *model.DormantServer
	if err := s.db.Transaction(func(tx *gorm.DB) error {
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

func (s Store) MakeServerArchived(ctx context.Context, id uuid.UUID) (*model.ArchivedServer, error) {
	dormant, err := s.GetDormantServer(ctx, id)
	if err != nil {
		return nil, err
	}

	var server *model.ArchivedServer
	if err := s.db.Transaction(func(tx *gorm.DB) error {
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
	return s.GetArchivedServer(ctx, server.ID)
}
