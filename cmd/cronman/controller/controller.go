// Package controller is responsible for monitoring server definitions, launching
// new servers, and shutting down active servers.
package controller

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/db"
	ierrors "github.com/tjper/rustcron/cmd/cronman/errors"
	"github.com/tjper/rustcron/cmd/cronman/logger"
	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/cmd/cronman/userdata"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

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

	dormant := &model.DormantServer{
		Server: input,
	}
	if err := ctrl.creator.Create(ctx, dormant); err != nil {
		return nil, fmt.Errorf("while creating dormant server: %w", err)
	}

	if err := ctrl.notifier.Notify(ctx); err != nil {
		return nil, fmt.Errorf("while notifying director: %w", err)
	}

	return dormant, nil
}

// GetServer retrieves the server from the underlyig store. The returned
// interface{} may be a model.LiveServer or a model.DormantServer. If a
// server has been archived or DNE, the interface{} will be nil and a
// 2nd return value of ErrServerDNE will be returned.
func (ctrl Controller) GetServer(
	ctx context.Context,
	id uuid.UUID,
) (interface{}, error) {
	liveServer, err := ctrl.store.GetLiveServer(ctx, id)
	if err == nil {
		return liveServer, nil
	}
	if err != nil && !errors.Is(err, ierrors.ErrServerNotLive) {
		return nil, err
	}

	dormantServer, err := ctrl.store.GetDormantServer(ctx, id)
	if errors.Is(err, ierrors.ErrServerNotDormant) {
		return nil, ierrors.ErrServerDNE
	}
	if err != nil {
		return nil, err
	}
	return dormantServer, nil
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

	find := db.FindDormantServer{
		ServerID: id,
	}
	if err := ctrl.finder.Find(ctx, &find); err != nil {
		return nil, fmt.Errorf("while starting server: %w", err)
	}
	dormant := find.Result

	server := find.Result.Server
	options := []userdata.Option{
		userdata.WithQueueBypassPlugin(),
		userdata.WithUserCfg(server.ID.String(), server.Moderators.SteamIDs()),
		userdata.WithServerCfg(server.ID.String(), server.Vips.Active().SteamIDs()),
	}

	wipe := server.Wipes.CurrentWipe()
	if !wipe.AppliedAt.Valid {
		switch wipe.Kind {
		case model.WipeKindMap:
			options = append(options, userdata.WithMapWipe(server.ID.String()))
		case model.WipeKindFull:
			options = append(options, userdata.WithMapWipe(server.ID.String()))
			options = append(options, userdata.WithBluePrintWipe(server.ID.String()))
		}
	}

	if err := ctrl.serverController.Region(dormant.Server.Region).StartInstance(
		ctx,
		dormant.Server.InstanceID,
		server.Userdata(options...),
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

	// Assumes wipe is being applied as part of the userdata to the StartInstance
	// call.
	if !wipe.AppliedAt.Valid {
		update := db.UpdateWipeApplied{
			WipeID: wipe.ID,
		}
		if err := ctrl.execer.Exec(ctx, update); err != nil {
			return nil, fmt.Errorf("while updating server wipe: %w", err)
		}
	}

	find = db.FindDormantServer{
		ServerID: id,
	}
	if err := ctrl.finder.Find(ctx, &find); err != nil {
		return nil, fmt.Errorf("while finding started dormant server: %w", err)
	}

	return &find.Result, nil
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

// WipeServer wipes the specified server.
func (ctrl *Controller) WipeServer(ctx context.Context, id uuid.UUID, wipe model.Wipe) error {
	create := db.CreateWipe{
		ServerID: id,
		Wipe:     wipe,
	}
	if err := ctrl.execer.Exec(ctx, create); err != nil {
		return fmt.Errorf("while wiping server: %w", err)
	}
	return nil
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

	if err := ctrl.store.ListServers(ctx, dst); err != nil {
		return err
	}

	return nil
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

// LiveServersRconForEach executes the specified function for each live server.
func (ctrl *Controller) LiveServerRconForEach(
	ctx context.Context,
	fn func(context.Context, model.LiveServer, rcon.IRcon) error,
) error {
	var servers model.LiveServers
	if err := ctrl.finder.Find(ctx, &servers); err != nil {
		return fmt.Errorf("while listing live servers: %w", err)
	}

	closure := func(server model.LiveServer) {
		client, err := ctrl.hub.Dial(
			ctx,
			fmt.Sprintf("%s:28016", server.Server.ElasticIP),
			server.Server.RconPassword,
		)
		if err != nil {
			ctrl.logger.Error(
				"while dialing rcon",
				zap.Stringer("server", server.Server.ID),
				zap.Error(err),
			)
			return
		}
		defer client.Close()

		if err := fn(ctx, server, client); err != nil {
			ctrl.logger.Error(
				"while executing live server fn",
				zap.Stringer("server", server.Server.ID),
				zap.Error(err),
			)
		}
	}

	for _, server := range servers {
		closure(server)
	}
	return nil
}

// CaptureServerInfo retrieves and stores the server info specified live server.
func (ctrl *Controller) CaptureServerInfo(ctx context.Context, server model.LiveServer, rcon rcon.IRcon) error {
	serverInfo, err := rcon.ServerInfo(ctx)
	if err != nil {
		return fmt.Errorf("while retrieving server info via rcon: %w", err)
	}

	update := db.UpdateLiveServerInfo{
		LiveServerID: server.ID,
		Changes: map[string]interface{}{
			"active_players": serverInfo.Players,
			"queued_players": serverInfo.Queued,
		},
	}

	err = ctrl.execer.Exec(ctx, update)
	if errors.Is(err, db.ErrServerNotLive) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("while updating server server info: %w", err)
	}
	return nil
}

func (ctrl *Controller) SayServerTimeRemaining(ctx context.Context, server model.LiveServer, rcon rcon.IRcon) error {
	_, when, err := server.Server.Events.NextEvent(ctrl.time.Now(), model.EventKindStop)
	if err != nil {
		return fmt.Errorf("while determining next live server event: %w", err)
	}

	until := ctrl.time.Until(*when)
	until = until.Round(time.Minute)

	var b strings.Builder
	fmt.Fprintf(&b, "%s will be going offline in", server.Server.Name)

	hours := int(until.Hours())
	if hours > 1 {
		fmt.Fprintf(&b, " %d hours", hours)
	} else if hours > 0 {
		fmt.Fprintf(&b, " %d hour", hours)
	}

	minutes := int(until.Minutes()) - (hours * 60)
	if hours > 0 && minutes > 0 {
		fmt.Fprintf(&b, " and")
	}
	if minutes > 1 {
		fmt.Fprintf(&b, " %d minutes", minutes)
	} else if minutes > 0 {
		fmt.Fprintf(&b, " %d minute", minutes)
	}

	fmt.Fprint(&b, ". Please visit rustpm.com for more scheduling information, an overview of our servers, and VIP access!")

	if err := rcon.Say(ctx, b.String()); err != nil {
		return fmt.Errorf("while saying server time remaining: %w", err)
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
	pingctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	if err := ctrl.waiter.UntilReady(
		pingctx,
		rconURL(ip, password),
	); err != nil {
		return fmt.Errorf("unable to ping server instance; %w", err)
	}
	return nil
}

// --- helpers ---

func rconURL(host, password string) string {
	url := url.URL{
		Scheme: "ws",
		Host:   net.JoinHostPort(host, "28016"),
		Path:   password,
	}
	return url.String()
}
