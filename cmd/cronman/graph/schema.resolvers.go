package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/google/uuid"
	dbmodel "github.com/tjper/rustcron/cmd/cronman/db/model"
	"github.com/tjper/rustcron/cmd/cronman/graph/generated"
	graphmodel "github.com/tjper/rustcron/cmd/cronman/graph/model"
	"github.com/tjper/rustcron/cmd/cronman/logger"
	"go.uber.org/zap"
)

func (r *mutationResolver) CreateServer(ctx context.Context, input graphmodel.NewServer) (*graphmodel.CreateServerResult, error) {
	logger := r.logger.With(logger.ContextFields(ctx)...)

	server, err := r.ctrl.CreateServer(ctx, newDbDefinition(input))
	if err != nil {
		logger.Error("error creating server", zap.Error(err))
		return nil, errInternalServer
	}

	return &graphmodel.CreateServerResult{
		Server: newModelDormantServer(*server),
	}, nil
}

func (r *mutationResolver) ArchiveServer(ctx context.Context, id string) (*graphmodel.ArchiveServerResult, error) {
	logger := r.logger.With(logger.ContextFields(ctx)...)

	serverID, err := uuid.Parse(id)
	if err != nil {
		logger.Error("invalid UUID", zap.String("uuid", id), zap.Error(err))
		return nil, errInvalidUUID
	}

	server, err := r.ctrl.ArchiveServer(ctx, serverID)
	if err != nil {
		logger.Error("error archiving server", zap.Error(err))
		return nil, errInternalServer
	}

	return &graphmodel.ArchiveServerResult{
		Server: newModelArchivedServer(*server),
	}, nil
}

func (r *mutationResolver) UpdateServer(ctx context.Context, id string, changes map[string]interface{}) (*graphmodel.UpdateServerResult, error) {
	logger := r.logger.With(logger.ContextFields(ctx)...)

	serverId, err := uuid.Parse(id)
	if err != nil {
		logger.Error("invalid UUID", zap.String("uuid", id), zap.Error(err))
		return nil, errInvalidUUID
	}

	definition, err := r.ctrl.UpdateServer(
		ctx,
		serverId,
		changes,
	)
	if err != nil {
		logger.Error("unable to update server definition", zap.Error(err))
		return nil, errInternalServer
	}
	return &graphmodel.UpdateServerResult{
		Definition: newModelServerDefinition(*definition),
	}, nil
}

func (r *mutationResolver) StartServer(ctx context.Context, id string) (*graphmodel.StartServerResult, error) {
	logger := r.logger.With(logger.ContextFields(ctx)...)

	dormantServerID, err := uuid.Parse(id)
	if err != nil {
		logger.Error("invalid UUID", zap.String("uuid", id), zap.Error(err))
		return nil, errInvalidUUID
	}

	if _, err := r.ctrl.StartServer(ctx, dormantServerID); err != nil {
		logger.Error("unable to start server", zap.Error(err))
		return nil, errInternalServer
	}
	liveServer, err := r.ctrl.MakeServerLive(ctx, dormantServerID)
	if err != nil {
		logger.Error("unable to transition server to live", zap.Error(err))
		return nil, errInternalServer
	}

	return &graphmodel.StartServerResult{
		Server: newModelLiveServer(*liveServer),
	}, nil
}

func (r *mutationResolver) StopServer(ctx context.Context, id string) (*graphmodel.StopServerResult, error) {
	logger := r.logger.With(logger.ContextFields(ctx)...)

	serverId, err := uuid.Parse(id)
	if err != nil {
		logger.Error("invalid UUID", zap.String("uuid", id), zap.Error(err))
		return nil, errInvalidUUID
	}

	dormantServer, err := r.ctrl.StopServer(ctx, serverId)
	if err != nil {
		logger.Error("error stopping server", zap.Error(err))
		return nil, errInternalServer
	}

	return &graphmodel.StopServerResult{
		Server: newModelDormantServer(*dormantServer),
	}, nil
}

func (r *mutationResolver) AddServerModerators(ctx context.Context, id string, mods []*graphmodel.NewModerator) (*graphmodel.AddServerModeratorsResult, error) {
	logger := r.logger.With(logger.ContextFields(ctx)...)

	definitionID, err := uuid.Parse(id)
	if err != nil {
		logger.Error("invalid UUID", zap.String("uuid", id), zap.Error(err))
		return nil, errInvalidUUID
	}

	definition, err := r.ctrl.AddServerModerators(
		ctx,
		definitionID,
		newDbDefinitionModerators(definitionID, mods),
	)
	if err != nil {
		logger.Error("error adding server moderators", zap.Error(err))
		return nil, errInternalServer
	}

	return &graphmodel.AddServerModeratorsResult{
		Definition: newModelServerDefinition(*definition),
	}, nil
}

func (r *mutationResolver) RemoveServerModerators(ctx context.Context, id string, moderatorIds []string) (*graphmodel.RemoveServerModeratorsResult, error) {
	logger := r.logger.With(logger.ContextFields(ctx)...)

	definitionID, err := uuid.Parse(id)
	if err != nil {
		logger.Error("invalid UUID", zap.String("uuid", id), zap.Error(err))
		return nil, errInvalidUUID
	}

	moderatorUUIDs, err := toUUIDs(moderatorIds)
	if err != nil {
		logger.Error("invalid moderator UUID", zap.Strings("uuid", moderatorIds), zap.Error(err))
		return nil, errInvalidUUID
	}

	definition, err := r.ctrl.RemoveServerModerators(
		ctx,
		definitionID,
		moderatorUUIDs,
	)
	if err != nil {
		logger.Error("error removing server moderators", zap.Error(err))
		return nil, errInternalServer
	}

	return &graphmodel.RemoveServerModeratorsResult{
		Definition: newModelServerDefinition(*definition),
	}, nil
}

func (r *mutationResolver) AddServerTags(ctx context.Context, id string, tags []*graphmodel.NewTag) (*graphmodel.AddServerTagsResult, error) {
	logger := r.logger.With(logger.ContextFields(ctx)...)

	definitionID, err := uuid.Parse(id)
	if err != nil {
		logger.Error("invalid UUID", zap.String("uuid", id), zap.Error(err))
		return nil, errInvalidUUID
	}

	definition, err := r.ctrl.AddServerTags(
		ctx,
		definitionID,
		newDbDefinitionTags(definitionID, tags),
	)
	if err != nil {
		logger.Error("error removing tags from server definition", zap.Error(err))
		return nil, errInternalServer
	}

	return &graphmodel.AddServerTagsResult{
		Definition: newModelServerDefinition(*definition),
	}, nil
}

func (r *mutationResolver) RemoveServerTags(ctx context.Context, id string, tagIds []string) (*graphmodel.RemoveServerTagsResult, error) {
	logger := r.logger.With(logger.ContextFields(ctx)...)

	definitionID, err := uuid.Parse(id)
	if err != nil {
		logger.Error("invalid server UUID", zap.String("uuid", id), zap.Error(err))
		return nil, errInvalidUUID
	}

	tagUUIDs, err := toUUIDs(tagIds)
	if err != nil {
		logger.Error("invalid tag UUID", zap.String("uuid", id), zap.Error(err))
		return nil, errInvalidUUID
	}

	definition, err := r.ctrl.RemoveServerTags(ctx, definitionID, tagUUIDs)
	if err != nil {
		logger.Error("error removing tags from server definition", zap.Error(err))
		return nil, errInternalServer
	}

	return &graphmodel.RemoveServerTagsResult{
		Definition: newModelServerDefinition(*definition),
	}, nil
}

func (r *mutationResolver) AddServerEvents(ctx context.Context, id string, events []*graphmodel.NewEvent) (*graphmodel.AddServerEventsResult, error) {
	logger := r.logger.With(logger.ContextFields(ctx)...)

	definitionID, err := uuid.Parse(id)
	if err != nil {
		logger.Error("invalid UUID", zap.String("uuid", id), zap.Error(err))
		return nil, errInvalidUUID
	}

	definition, err := r.ctrl.AddServerEvents(
		ctx,
		definitionID,
		newDbDefinitionEvents(definitionID, events),
	)
	if err != nil {
		logger.Error("error adding events to server definition", zap.Error(err))
		return nil, err
	}

	return &graphmodel.AddServerEventsResult{
		Definition: newModelServerDefinition(*definition),
	}, nil
}

func (r *mutationResolver) RemoveServerEvents(ctx context.Context, id string, eventIds []string) (*graphmodel.RemoveServerEventsResult, error) {
	logger := r.logger.With(logger.ContextFields(ctx)...)

	definitionID, err := uuid.Parse(id)
	if err != nil {
		logger.Error("invalid server UUID", zap.String("uuid", id), zap.Error(err))
		return nil, errInvalidUUID
	}

	eventUUIDs, err := toUUIDs(eventIds)
	if err != nil {
		logger.Error("invalid event UUID", zap.String("uuid", id), zap.Error(err))
		return nil, errInvalidUUID
	}

	definition, err := r.ctrl.RemoveServerEvents(
		ctx,
		definitionID,
		eventUUIDs,
	)
	if err != nil {
		logger.Error("error adding events to server definition", zap.Error(err))
		return nil, err
	}

	return &graphmodel.RemoveServerEventsResult{
		Definition: newModelServerDefinition(*definition),
	}, nil
}

func (r *queryResolver) Servers(ctx context.Context, input graphmodel.ServersCriteria) (*graphmodel.ServersResult, error) {
	logger := r.logger.With(logger.ContextFields(ctx)...)

	servers := make([]graphmodel.Server, 0)
	switch input.State {
	case graphmodel.StateActive:
		liveServers := make([]dbmodel.LiveServer, 0)
		if err := r.ctrl.ListServers(ctx, &liveServers); err != nil {
			logger.Error("error listing live servers", zap.Error(err))
			return nil, errInternalServer
		}
		dormantServers := make([]dbmodel.DormantServer, 0)
		if err := r.ctrl.ListServers(ctx, &dormantServers); err != nil {
			logger.Error("error listing dormant servers", zap.Error(err))
			return nil, errInternalServer
		}
		servers = append(servers, newModelServers(liveServers)...)
		servers = append(servers, newModelServers(dormantServers)...)

	case graphmodel.StateArchived:
		archivedServers := make([]dbmodel.ArchivedServer, 0)
		if err := r.ctrl.ListServers(ctx, &archivedServers); err != nil {
			logger.Error("error listing archived servers", zap.Error(err))
			return nil, errInternalServer
		}
		servers = append(servers, newModelServers(archivedServers)...)
	}

	return &graphmodel.ServersResult{
		Servers: servers,
	}, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
