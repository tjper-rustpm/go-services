// Package controller is responsible for monitoring server definitions, launching
// new servers, and shutting down active servers.
package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/logger"
	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
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
	input model.Server,
) (*model.DormantServer, error) {
	instance, err := ctrl.serverController.Region(input.Region).CreateInstance(
		ctx,
		input.InstanceKind,
	)
	if err != nil {
		return nil, fmt.Errorf("creating instance; %w", err)
	}

	input.InstanceID = *instance.Instance.InstanceId
	input.AllocationID = *instance.Address.AllocationId
	input.ElasticIP = *instance.Address.PublicIp

	dormant := model.DormantServer{Server: input}
	if err := ctrl.store.Create(ctx, &dormant); err != nil {
		return nil, fmt.Errorf("creating server; %w", err)
	}

	if err := ctrl.notifier.Notify(ctx); err != nil {
		return nil, fmt.Errorf("notifying director; %w", err)
	}

	return &dormant, nil
}

type UpdateServerInput struct {
	ID      uuid.UUID
	Changes map[string]interface{}
}

// UpdateServer instructs the Controller to updates the server passed with the
// associated data.
func (ctrl Controller) UpdateServer(
	ctx context.Context,
	input UpdateServerInput,
) (*model.DormantServer, error) {
	dormant, err := ctrl.store.UpdateServer(ctx, input.ID, input.Changes)
	if err != nil {
		return nil, fmt.Errorf("update server; %w", err)
	}

	if err := ctrl.notifier.Notify(ctx); err != nil {
		return nil, fmt.Errorf("notifying director; %w", err)
	}

	return dormant, nil
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

	dormant, err := ctrl.store.GetDormantServer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get dormant server; %w", err)
	}

	userdata := dormant.Server.Userdata(
		userdata.WithQueueBypassPlugin(),
		userdata.WithUserCfg(dormant.Server.Moderators.SteamIDs()),
	)

	if err := ctrl.serverController.Region(dormant.Server.Region).StartInstance(
		ctx,
		dormant.Server.InstanceID,
		userdata,
	); err != nil {
		return nil, fmt.Errorf("start server instance; %w", err)
	}

	association, err := ctrl.serverController.Region(dormant.Server.Region).MakeInstanceAvailable(
		ctx,
		dormant.Server.InstanceID,
		dormant.Server.AllocationID,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to make server instance available; %w", err)
	}
	defer func() {
		if err := ctrl.serverController.Region(dormant.Server.Region).MakeInstanceUnavailable(
			ctx,
			*association.AssociationId,
		); err != nil {
			logger.Error("unable to make server instance unavailable", zap.Error(err))
		}
	}()

	if err := ctrl.pingUntilReady(
		ctx,
		dormant.Server.ElasticIP,
		dormant.Server.RconPassword,
	); err != nil {
		return nil, fmt.Errorf("unable to ping server instance; %w", err)
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
		return nil, fmt.Errorf("get dormant server; %w", err)
	}

	instance, err := ctrl.serverController.Region(server.Server.Region).MakeInstanceAvailable(
		ctx,
		server.Server.InstanceID,
		server.Server.AllocationID,
	)
	if err != nil {
		return nil, fmt.Errorf("make server instance available; %w", err)
	}
	if err := ctrl.pingUntilReady(
		ctx,
		server.Server.ElasticIP,
		server.Server.RconPassword,
	); err != nil {
		return nil, fmt.Errorf("ping until ready; %w", err)
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
		fmt.Sprintf("%s:28016", server.Server.ElasticIP),
		server.Server.RconPassword,
	)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := client.Quit(ctx); err != nil {
		return nil, err
	}

	if err := ctrl.serverController.Region(server.Server.Region).MakeInstanceUnavailable(
		ctx,
		server.AssociationID,
	); err != nil {
		return nil, err
	}
	if err := ctrl.serverController.Region(server.Server.Region).StopInstance(
		ctx,
		server.Server.InstanceID,
	); err != nil {
		return nil, err
	}
	return dormantServer, nil
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

func (ctrl *Controller) AddServerTags(
	ctx context.Context,
	serverID uuid.UUID,
	tags model.Tags,
) error {
	if _, err := ctrl.store.GetServer(ctx, serverID); err != nil {
		return fmt.Errorf("get server; serverID: %s, error: %w", serverID, err)
	}

	for i := range tags {
		tags[i].ServerID = serverID
	}

	if err := ctrl.store.Create(ctx, tags); err != nil {
		return fmt.Errorf("create server tags; serverID: %s, error: %w", serverID, err)
	}
	return nil
}

func (ctrl *Controller) RemoveServerTags(
	ctx context.Context,
	serverID uuid.UUID,
	tagIDs []uuid.UUID,
) error {
	if _, err := ctrl.store.GetServer(ctx, serverID); err != nil {
		return fmt.Errorf("get server; serverID: %s, error: %w", serverID, err)
	}

	if err := ctrl.store.Delete(ctx, &model.Tag{}, tagIDs); err != nil {
		return fmt.Errorf("delete server tags; serverID: %s, error: %w", serverID, err)
	}
	return nil
}

func (ctrl *Controller) AddServerEvents(
	ctx context.Context,
	serverID uuid.UUID,
	events model.Events,
) error {
	if _, err := ctrl.store.GetServer(ctx, serverID); err != nil {
		return fmt.Errorf("get server; serverID: %s, error: %w", serverID, err)
	}

	for i := range events {
		events[i].ServerID = serverID
	}

	if err := ctrl.store.Create(ctx, events); err != nil {
		return fmt.Errorf("create server events; serverID: %s, error: %w", serverID, err)
	}
	return nil
}

func (ctrl *Controller) RemoveServerEvents(
	ctx context.Context,
	serverID uuid.UUID,
	eventIDs []uuid.UUID,
) error {
	if _, err := ctrl.store.GetServer(ctx, serverID); err != nil {
		return fmt.Errorf("get server; serverID: %s, error: %w", serverID, err)
	}

	if err := ctrl.store.Delete(ctx, &model.Event{}, eventIDs); err != nil {
		return fmt.Errorf("delete server events; serverID: %s, error: %w", serverID, err)
	}
	return nil
}

func (ctrl *Controller) AddServerModerators(
	ctx context.Context,
	serverID uuid.UUID,
	moderators model.Moderators,
) error {
	server, err := ctrl.store.GetServer(ctx, serverID)
	if err != nil {
		return fmt.Errorf("get server; serverID: %s, error: %w", serverID, err)
	}

	for i := range moderators {
		moderators[i].ServerID = serverID
	}

	if server.StateType == model.LiveServerState {
		if err := ctrl.rconAddServerModerators(
			ctx,
			server.ElasticIP,
			server.RconPassword,
			moderators,
		); err != nil {
			return err
		}
	}

	if err := ctrl.store.Create(ctx, moderators); err != nil {
		return fmt.Errorf("create server moderators; serverID: %s, error: %w", serverID, err)
	}

	return nil
}

func (ctrl *Controller) RemoveServerModerators(
	ctx context.Context,
	serverID uuid.UUID,
	moderatorIDs []uuid.UUID,
) error {
	server, err := ctrl.store.GetServer(ctx, serverID)
	if err != nil {
		return fmt.Errorf("get server; serverID: %s, error: %w", serverID, err)
	}

	var moderators model.Moderators
	if err := ctrl.store.Find(ctx, moderators, moderatorIDs); err != nil {
		return fmt.Errorf(
			"find moderators; serverID: %s, moderatorIDs: %v, error: %w",
			serverID,
			moderatorIDs,
			err,
		)
	}

	if server.StateType == model.LiveServerState {
		if err := ctrl.rconRemoveServerModerators(
			ctx,
			server.ElasticIP,
			server.RconPassword,
			moderators,
		); err != nil {
			return err
		}
	}

	if err := ctrl.store.Delete(ctx, &model.Moderator{}, moderatorIDs); err != nil {
		return fmt.Errorf("delete server moderators; serverID: %s, error: %w", serverID, err)
	}

	return nil
}

// --- private ---

func (ctrl *Controller) rconAddServerModerators(
	ctx context.Context,
	elasticIP string,
	password string,
	moderators model.Moderators,
) error {
	logger := ctrl.logger.With(logger.ContextFields(ctx)...)

	client, err := ctrl.hub.Dial(
		ctx,
		fmt.Sprintf("%s:28016", elasticIP),
		password,
	)
	if err != nil {
		return fmt.Errorf("dial rcon; %w", err)
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

func (ctrl Controller) rconRemoveServerModerators(
	ctx context.Context,
	elasticIP string,
	password string,
	moderators model.Moderators,
) error {
	logger := ctrl.logger.With(logger.ContextFields(ctx)...)

	client, err := ctrl.hub.Dial(
		ctx,
		fmt.Sprintf("%s:28016", elasticIP),
		password,
	)
	if err != nil {
		return fmt.Errorf("dial rcon; %w", err)
	}
	defer client.Close()

	for _, moderator := range moderators {
		if err := client.RemoveModerator(
			ctx,
			moderator.SteamID,
		); err != nil && !errors.Is(err, rcon.ErrModeratorExists) {
			logger.Error("remove moderators", zap.Error(err))
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

func rconURL(url, password string) string {
	return fmt.Sprintf("ws://%s:28016/%s", url, password)
}
