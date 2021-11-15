// Package controller is responsible for monitoring server definitions, launching
// new servers, and shutting down active servers.
package controller

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/logger"
	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/cmd/cronman/schedule"
	"github.com/tjper/rustcron/cmd/cronman/userdata"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WatchAndDirect instructs the Controller to collect upcoming server events and
// pass them to the EventsProcessor.
func (ctrl Controller) WatchAndDirect(ctx context.Context) error {
	// acquire distributed lock, only one instance runs the controller
	if err := ctrl.distributedLock.Lock(ctx); err != nil {
		return fmt.Errorf("failed acquire controller lock; %w", err)
	}
	defer ctrl.distributedLock.Unlock(ctx)

	ctrl.logger.Info("subscribed to refresh subject")
	sub := ctrl.redis.Subscribe(ctx, refreshSubj)
	defer func() {
		if err := sub.Close(); err != nil {
			ctrl.logger.Error("failed to close refresh subscription")
		}
	}()

	for {
		ctrl.logger.Info("retrieving server events")
		_, err := ctrl.store.ListActiveServerEvents(ctx)
		if err != nil {
			return fmt.Errorf("failed to list events; %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sub.Channel():
			continue
		}
	}
}

// CreateServer instruct the Controller to create the server based on the input
// specified. On success, the server has been created and is in a dormant
// state.
func (ctrl Controller) CreateServer(
	ctx context.Context,
	input model.ServerDefinition,
) (*model.DormantServer, error) {
	instance, err := ctrl.serverController.Region(input.Region).CreateInstance(
		ctx,
		input.InstanceKind,
	)
	if err != nil {
		return nil, err
	}
	input.InstanceID = *instance.Instance.InstanceId
	input.AllocationID = *instance.Address.AllocationId
	input.ElasticIP = *instance.Address.PublicIp

	// TODO: Merge CreateDefinition and Create DormantServer into single
	// transaction.
	definition, err := ctrl.store.CreateDefinition(ctx, input)
	if err != nil {
		return nil, err
	}

	dormant := &model.DormantServer{
		ServerDefinitionID: definition.ID,
	}
	if err := ctrl.store.Create(ctx, dormant); err != nil {
		return nil, err
	}

	if err := ctrl.notifier.Notify(ctx); err != nil {
		return nil, err
	}

	return ctrl.store.GetDormantServer(ctx, dormant.ID)
}

// ArchiveServer instruct the Controller to archive the server specified by id.
// On success, the server has been moved to the archived state. It will
// no longer show in active server lists.
func (ctrl Controller) ArchiveServer(
	ctx context.Context,
	id uuid.UUID,
) (*model.ArchivedServer, error) {
	server, err := ctrl.store.MakeServerArchived(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := ctrl.notifier.Notify(ctx); err != nil {
		return nil, err
	}
	return server, nil
}

// StartServer instructs the Controller start the server specified by id. Once
// the method returns successfully, the server has been updated, and is
// running, but has not yet been exposed to users.
func (ctrl Controller) StartServer(
	ctx context.Context,
	id uuid.UUID,
) (*model.DormantServer, error) {
	logger := ctrl.logger.With(logger.ContextFields(ctx)...)

	server, err := ctrl.store.GetDormantServer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("unable to get dormant server; %w", err)
	}

	userdata, err := generateUserData(server.ServerDefinition)
	if err != nil {
		return nil, fmt.Errorf("unable to generate server userdata; %w", err)
	}
	if err := ctrl.serverController.Region(server.ServerDefinition.Region).StartInstance(
		ctx,
		server.ServerDefinition.InstanceID,
		userdata,
	); err != nil {
		return nil, fmt.Errorf("unable to start server instance; %w", err)
	}

	association, err := ctrl.serverController.Region(server.ServerDefinition.Region).MakeInstanceAvailable(
		ctx,
		server.ServerDefinition.InstanceID,
		server.ServerDefinition.AllocationID,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to make server instance available; %w", err)
	}
	defer func() { // make unavailable on func return
		if err := ctrl.serverController.Region(server.ServerDefinition.Region).MakeInstanceUnavailable(
			ctx,
			*association.AssociationId,
		); err != nil {
			logger.Error("unable to make server instance unavailable", zap.Error(err))
		}
	}()

	if err := ctrl.pingUntilReady(
		ctx,
		server.ServerDefinition.ElasticIP,
		server.ServerDefinition.RconPassword,
	); err != nil {
		return nil, fmt.Errorf("unable to ping server instance; %w", err)
	}

	if err := ctrl.removeModeratorsPendingRemoval(
		ctx,
		server.ServerDefinition.ID,
		server.ServerDefinition.ElasticIP,
		server.ServerDefinition.RconPassword,
	); err != nil {
		return nil, fmt.Errorf("unable to remove moderators pending removal; %w", err)
	}
	if err := ctrl.rconAddServerModerators(
		ctx,
		server.ServerDefinition.ElasticIP,
		server.ServerDefinition.RconPassword,
		server.ServerDefinition.Moderators,
	); err != nil {
		return nil, fmt.Errorf("unable to add moderators; %w", err)
	}
	return ctrl.store.GetDormantServer(ctx, id)
}

// MakeServerLive instructs the Controller to make the server specified by the id
// live.
func (ctrl Controller) MakeServerLive(
	ctx context.Context,
	id uuid.UUID,
) (*model.LiveServer, error) {
	server, err := ctrl.store.GetDormantServer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("unable to get dormant server; %w", err)
	}

	instance, err := ctrl.serverController.Region(server.ServerDefinition.Region).MakeInstanceAvailable(
		ctx,
		server.ServerDefinition.InstanceID,
		server.ServerDefinition.AllocationID,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to make server instance available; %w", err)
	}
	if err := ctrl.pingUntilReady(
		ctx,
		server.ServerDefinition.ElasticIP,
		server.ServerDefinition.RconPassword,
	); err != nil {
		return nil, fmt.Errorf("unable to ping server instance; %w", err)
	}

	return ctrl.store.MakeServerLive(
		ctx,
		db.MakeServerLiveInput{
			ID:            id,
			AssociationID: *instance.AssociationId,
		},
	)
}

// StopServer instructs the Controller stop the server specified by id. Once the
// method returns successfully, the server has been stopped.
func (ctrl *Controller) StopServer(ctx context.Context, id uuid.UUID) (*model.DormantServer, error) {
	server, err := ctrl.store.GetLiveServer(ctx, id)
	if err != nil {
		return nil, err
	}
	dormantServer, err := ctrl.store.MakeServerDormant(ctx, id)
	if err != nil {
		return nil, err
	}

	client, err := ctrl.hub.Dial(
		ctx,
		fmt.Sprintf("%s:28016", server.ServerDefinition.ElasticIP),
		server.ServerDefinition.RconPassword,
	)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := client.Quit(ctx); err != nil {
		return nil, err
	}

	if err := ctrl.serverController.Region(server.ServerDefinition.Region).MakeInstanceUnavailable(
		ctx,
		server.AssociationID,
	); err != nil {
		return nil, err
	}
	if err := ctrl.serverController.Region(server.ServerDefinition.Region).StopInstance(
		ctx,
		server.ServerDefinition.InstanceID,
	); err != nil {
		return nil, err
	}
	return dormantServer, nil
}

// UpdateServer reconfigures the server as specified by changes. On success,
// the server has been reconfigured.
func (ctrl *Controller) UpdateServer(
	ctx context.Context,
	id uuid.UUID,
	changes map[string]interface{},
) (*model.ServerDefinition, error) {
	definition, err := ctrl.store.UpdateServerDefinition(ctx, id, changes)
	if err != nil {
		return nil, err
	}
	return definition, ctrl.notifier.Notify(ctx)
}

// AddServerModerators adds the specified moderators to the passed server
// definition. If the server is live, the moderators are add as part of this
// call. Otherwise, the moderators will be added when the server goes live.
func (ctrl *Controller) AddServerModerators(
	ctx context.Context,
	definitionID uuid.UUID,
	moderators model.DefinitionModerators,
) (*model.ServerDefinition, error) {
	definition, err := ctrl.store.GetDefinition(ctx, definitionID)
	if err != nil {
		return nil, err
	}

	if err := ctrl.store.Create(ctx, &moderators); err != nil {
		return nil, err
	}

	isLive, err := ctrl.store.DefinitionIsLive(ctx, definitionID)
	if err != nil {
		return nil, err
	}
	if isLive {
		if err := ctrl.rconAddServerModerators(
			ctx,
			definition.ElasticIP,
			definition.RconPassword,
			moderators,
		); err != nil {
			return nil, err
		}
	}

	return ctrl.store.GetDefinition(ctx, definitionID)
}

// RemoveServerModerators removes the specified moderators from the passed
// server definition. If the server is live, the moderators are removes as part
// of this call. Otherwise, the moderators will be removed when the server goes
// live.
func (ctrl *Controller) RemoveServerModerators(
	ctx context.Context,
	definitionID uuid.UUID,
	moderatorIDs []uuid.UUID,
) (*model.ServerDefinition, error) {

	definition, err := ctrl.store.GetDefinition(ctx, definitionID)
	if err != nil {
		return nil, err
	}

	if err := ctrl.markServerModeratorsForRemoval(ctx, moderatorIDs); err != nil {
		return nil, err
	}

	isLive, err := ctrl.store.DefinitionIsLive(ctx, definitionID)
	if err != nil {
		return nil, err
	}
	if isLive {
		if err := ctrl.removeModeratorsPendingRemoval(
			ctx,
			definitionID,
			definition.ElasticIP,
			definition.RconPassword,
		); err != nil {
			return nil, err
		}
	}
	return ctrl.store.GetDefinition(ctx, definitionID)
}

// AddServerTags adds the specified tags to the passed server definition.
func (ctrl *Controller) AddServerTags(
	ctx context.Context,
	definitionID uuid.UUID,
	tags model.DefinitionTags,
) (*model.ServerDefinition, error) {
	if _, err := ctrl.store.GetDefinition(ctx, definitionID); err != nil {
		return nil, err
	}
	if err := ctrl.store.Create(ctx, &tags); err != nil {
		return nil, err
	}
	return ctrl.store.GetDefinition(ctx, definitionID)
}

// RemoveServerTags removes the specified tags from the passed server
// definition.
func (ctrl *Controller) RemoveServerTags(
	ctx context.Context,
	definitionID uuid.UUID,
	tagIDs []uuid.UUID,
) (*model.ServerDefinition, error) {
	if _, err := ctrl.store.GetDefinition(ctx, definitionID); err != nil {
		return nil, err
	}

	for _, tagID := range tagIDs {
		if err := ctrl.store.Delete(ctx, &model.DefinitionTag{}, tagID); err != nil {
			return nil, err
		}
	}
	return ctrl.store.GetDefinition(ctx, definitionID)
}

// AddServerEvents adds the specified events to the passed server definition.
func (ctrl *Controller) AddServerEvents(
	ctx context.Context,
	definitionID uuid.UUID,
	events model.DefinitionEvents,
) (*model.ServerDefinition, error) {
	if _, err := ctrl.store.GetDefinition(ctx, definitionID); err != nil {
		return nil, err
	}
	if err := ctrl.store.Create(ctx, &events); err != nil {
		return nil, err
	}
	return ctrl.store.GetDefinition(ctx, definitionID)
}

// RemoveServerEvents removes the specified events from the passed server
// definition.
func (ctrl *Controller) RemoveServerEvents(
	ctx context.Context,
	definitionID uuid.UUID,
	eventIDs []uuid.UUID,
) (*model.ServerDefinition, error) {
	if _, err := ctrl.store.GetDefinition(ctx, definitionID); err != nil {
		return nil, err
	}

	for _, eventID := range eventIDs {
		if err := ctrl.store.Delete(ctx, &model.DefinitionEvent{}, eventID); err != nil {
			return nil, err
		}
	}
	return ctrl.store.GetDefinition(ctx, definitionID)
}

var errInvalidServerType = errors.New("invalid server type")

// ListServers evaluates the dst and populates it with the related data. The
// dst should be of type *model.LiveServer, *model.DormantServer,
// *model.ArchivedServer.
func (ctrl *Controller) ListServers(ctx context.Context, dst interface{}) error {
	switch dst.(type) {
	case *[]model.LiveServer:
	case *[]model.DormantServer:
	case *[]model.ArchivedServer:
		break
	default:
		return errInvalidServerType
	}
	return ctrl.store.ListServers(ctx, dst)
}

// --- private ---

func (ctrl *Controller) markServerModeratorsForRemoval(
	ctx context.Context,
	moderatorsIDs []uuid.UUID,
) error {
	for _, moderatorID := range moderatorsIDs {
		if err := ctrl.store.Update(
			ctx,
			&model.DefinitionModerator{Model: model.Model{ID: moderatorID}},
			&model.DefinitionModerator{QueuedDeletionAt: sql.NullTime{Time: time.Now(), Valid: true}},
		); err != nil {
			ctrl.logger.Error(
				"error marking moderator for removal",
				zap.String("moderatorID", moderatorID.String()),
			)
			return err
		}
	}
	return nil
}

func (ctrl *Controller) removeModeratorsPendingRemoval(
	ctx context.Context,
	definitionID uuid.UUID,
	elasticIP string,
	password string,
) error {
	logger := ctrl.logger.With(logger.ContextFields(ctx)...)

	// filter moderators that should be deleted
	moderators, err := ctrl.store.ListModeratorsPendingRemoval(ctx, definitionID)
	if err != nil {
		return err
	}

	client, err := ctrl.hub.Dial(
		ctx,
		fmt.Sprintf("%s:28016", elasticIP),
		password,
	)
	if err != nil {
		return fmt.Errorf("error dialing rcon definition; %w", err)
	}
	defer client.Close()

	for _, moderator := range moderators {
		if err := client.RemoveModerator(
			ctx,
			moderator.SteamID,
		); err != nil && !errors.Is(err, rcon.ErrModeratorDNE) {
			logger.Error("unable to remove moderator from definition", zap.Error(err))
		}
		if err := ctrl.store.Delete(ctx, &model.DefinitionModerator{}, moderator.ID); err != nil {
			logger.Error("unable to complete moderator removal", zap.Error(err))
		}
	}

	return nil
}

func (ctrl *Controller) rconAddServerModerators(
	ctx context.Context,
	elasticIP string,
	password string,
	moderators model.DefinitionModerators,
) error {

	logger := ctrl.logger.With(logger.ContextFields(ctx)...)

	client, err := ctrl.hub.Dial(
		ctx,
		fmt.Sprintf("%s:28016", elasticIP),
		password,
	)
	if err != nil {
		return fmt.Errorf("error dialing Rcon server; %w", err)
	}
	defer client.Close()

	for _, moderator := range moderators {
		if err := client.AddModerator(
			ctx,
			moderator.SteamID,
		); err != nil && !errors.Is(err, rcon.ErrModeratorExists) {
			logger.Error("unable to add moderators to server", zap.Error(err))
		}
	}
	return nil
}

// pingUntilReady pings the ip specified until it accepts the websocket
// connection. This may be done to ensure the specified ip is available before
// proceeding.
func (ctrl *Controller) pingUntilReady(ctx context.Context, ip, password string) error {
	pingctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	if err := ctrl.waiter.UntilReady(
		pingctx,
		rconURL(ip, password),
		time.Minute,
	); err != nil {
		return fmt.Errorf("unable to ping server instance; %w", err)
	}
	return nil
}

// --- helpers ---

func generateUserData(server model.ServerDefinition) (string, error) {
	fullWipe, err := schedule.IsFullWipe(
		server.BlueprintWipeFrequency,
		schedule.WipeDayToWeekDay(server.WipeDay),
		time.Now().UTC())
	if err != nil {
		return "", fmt.Errorf("unable to generate userdata; %w", err)
	}
	mapWipe, err := schedule.IsMapWipe(
		server.MapWipeFrequency,
		schedule.WipeDayToWeekDay(server.WipeDay),
		time.Now().UTC())
	if err != nil {
		return "", fmt.Errorf("unable to generate userdata; %w", err)
	}

	var opts []userdata.Option
	switch {
	case fullWipe:
		opts = []userdata.Option{userdata.WithBluePrintWipe(), userdata.WithMapWipe()}
	case mapWipe:
		opts = []userdata.Option{userdata.WithMapWipe()}
	default:
		opts = []userdata.Option{}
	}

	return userdata.Generate(
		server.Name,
		server.RconPassword,
		int(server.MaxPlayers),
		int(server.MapSize),
		int(server.MapSeed),
		int(server.MapSalt),
		int(server.TickRate),
		opts...,
	), nil
}

func rconURL(url, password string) string {
	return fmt.Sprintf("ws://%s:28016/%s", url, password)
}
