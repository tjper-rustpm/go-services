package graph

import (
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/tjper/rustcron/cmd/cronman/db/model"
	"github.com/tjper/rustcron/cmd/cronman/graph/model"
	graphmodel "github.com/tjper/rustcron/cmd/cronman/graph/model"
)

func newDbDefinition(
	server graphmodel.NewServer,
) dbmodel.ServerDefinition {
	return dbmodel.ServerDefinition{
		Name:                   server.Name,
		InstanceKind:           server.InstanceKind,
		MaxPlayers:             uint16(server.MaxPlayers),
		MapSize:                uint16(server.MapSize),
		MapSeed:                uint16(server.MapSeed),
		MapSalt:                uint16(server.MapSalt),
		TickRate:               uint8(server.TickRate),
		RconPassword:           server.RconPassword,
		Description:            server.Description,
		Url:                    server.URL,
		Background:             server.Background,
		BannerUrl:              server.BannerURL,
		WipeDay:                server.WipeDay,
		BlueprintWipeFrequency: server.BlueprintWipeFrequency,
		MapWipeFrequency:       server.MapWipeFrequency,
		Region:                 server.Region,
		Tags:                   newDbDefinitionTags(uuid.Nil, server.Tags),
		Events:                 newDbDefinitionEvents(uuid.Nil, server.Schedule),
		Moderators:             newDbDefinitionModerators(uuid.Nil, server.Moderators),
	}
}

func newDbDefinitionEvents(
	serverDefinitionID uuid.UUID,
	events []*graphmodel.NewEvent,
) []dbmodel.DefinitionEvent {
	dbEvents := make([]dbmodel.DefinitionEvent, 0, len(events))
	for _, event := range events {
		dbEvents = append(dbEvents, dbmodel.DefinitionEvent{
			ServerDefinitionID: serverDefinitionID,
			Weekday:            time.Weekday(event.Day),
			Hour:               uint8(event.Hour),
			EventKind:          event.Kind,
		})
	}
	return dbEvents
}

func newDbDefinitionTags(
	serverDefinitionID uuid.UUID,
	tags []*graphmodel.NewTag,
) []dbmodel.DefinitionTag {
	dbTags := make([]dbmodel.DefinitionTag, 0, len(tags))
	for _, tag := range tags {
		dbTags = append(dbTags, dbmodel.DefinitionTag{
			ServerDefinitionID: serverDefinitionID,
			Description:        tag.Description,
			Icon:               tag.Icon,
			Value:              tag.Value,
		})
	}
	return dbTags
}

func newDbDefinitionModerators(
	serverDefinitionID uuid.UUID,
	moderators []*graphmodel.NewModerator,
) []dbmodel.DefinitionModerator {
	dbModerators := make([]dbmodel.DefinitionModerator, 0, len(moderators))
	for _, moderator := range moderators {
		dbModerators = append(dbModerators, dbmodel.DefinitionModerator{
			ServerDefinitionID: serverDefinitionID,
			SteamID:            moderator.SteamID,
		})
	}
	return dbModerators
}

func newModelServerDefinition(definition dbmodel.ServerDefinition) *graphmodel.ServerDefinition {
	return &graphmodel.ServerDefinition{
		ID:                     definition.ID.String(),
		Name:                   definition.Name,
		InstanceKind:           graphmodel.InstanceKind(definition.InstanceKind),
		InstanceID:             definition.InstanceID,
		AllocationID:           definition.AllocationID,
		ElasticIP:              definition.ElasticIP,
		MaxPlayers:             int(definition.MaxPlayers),
		MapSize:                int(definition.MapSize),
		MapSeed:                int(definition.MapSeed),
		MapSalt:                int(definition.MapSalt),
		TickRate:               int(definition.TickRate),
		RconPassword:           definition.RconPassword,
		Description:            definition.Description,
		Background:             graphmodel.BackgroundKind(definition.Background),
		URL:                    definition.Url,
		BannerURL:              definition.BannerUrl,
		WipeDay:                graphmodel.WipeDay(definition.WipeDay),
		BlueprintWipeFrequency: graphmodel.WipeFrequency(definition.BlueprintWipeFrequency),
		MapWipeFrequency:       graphmodel.WipeFrequency(definition.MapWipeFrequency),
		Region:                 graphmodel.Region(definition.Region),
		Schedule:               newModelSchedule(definition.Events),
		Moderators:             newModelModerators(definition.Moderators),
		Tags:                   newModelTags(definition.Tags),
	}
}

func newModelSchedule(events []dbmodel.DefinitionEvent) []*graphmodel.Event {
	schedule := make([]*graphmodel.Event, 0, len(events))
	for _, event := range events {
		schedule = append(schedule, &graphmodel.Event{
			ID:   event.ID.String(),
			Day:  int(event.Weekday),
			Hour: int(event.Hour),
			Kind: event.EventKind,
		})
	}
	return schedule
}

func newModelModerators(moderators []dbmodel.DefinitionModerator) []*graphmodel.Moderator {
	modelMods := make([]*graphmodel.Moderator, 0, len(moderators))
	for _, moderator := range moderators {
		modelMods = append(modelMods, &graphmodel.Moderator{
			ID:      moderator.ID.String(),
			SteamID: moderator.SteamID,
		})
	}
	return modelMods
}
func newModelTags(tags []dbmodel.DefinitionTag) []*graphmodel.Tag {
	modelTags := make([]*graphmodel.Tag, 0, len(tags))
	for _, tag := range tags {
		modelTags = append(modelTags, &graphmodel.Tag{
			ID:          tag.ID.String(),
			Description: tag.Description,
			Icon:        tag.Icon,
			Value:       tag.Value,
		})
	}
	return modelTags
}

func newModelServers(serversI interface{}) []model.Server {
	modelServers := make([]graphmodel.Server, 0)
	switch servers := serversI.(type) {
	case []dbmodel.LiveServer:
		modelServers = append(modelServers, newModelLiveServers(servers)...)
	case []dbmodel.DormantServer:
		modelServers = append(modelServers, newModelDormantServers(servers)...)
	case []dbmodel.ArchivedServer:
		modelServers = append(modelServers, newModelArchivedServers(servers)...)
	default:
		panic("invalid servers type")
	}
	return modelServers
}

func newModelLiveServers(servers []dbmodel.LiveServer) []graphmodel.Server {
	res := make([]graphmodel.Server, 0, len(servers))
	for _, server := range servers {
		res = append(res, newModelLiveServer(server))
	}
	return res
}

func newModelDormantServers(servers []dbmodel.DormantServer) []graphmodel.Server {
	res := make([]graphmodel.Server, 0, len(servers))
	for _, server := range servers {
		res = append(res, newModelDormantServer(server))
	}
	return res
}

func newModelArchivedServers(servers []dbmodel.ArchivedServer) []graphmodel.Server {
	res := make([]graphmodel.Server, 0, len(servers))
	for _, server := range servers {
		res = append(res, newModelArchivedServer(server))
	}
	return res
}

func newModelLiveServer(server dbmodel.LiveServer) *graphmodel.LiveServer {
	return &graphmodel.LiveServer{
		ID:            server.ID.String(),
		Definition:    newModelServerDefinition(server.ServerDefinition),
		AssociationID: server.AssociationID,
		ActivePlayers: int(server.ActivePlayers),
		QueuedPlayers: int(server.QueuedPlayers),
		UpdatedAt:     server.UpdatedAt,
		CreatedAt:     server.CreatedAt,
	}
}

func newModelDormantServer(server dbmodel.DormantServer) *graphmodel.DormantServer {
	return &graphmodel.DormantServer{
		ID:         server.ID.String(),
		Definition: newModelServerDefinition(server.ServerDefinition),
		StartsAt: server.ServerDefinition.Events.NextOf(
			time.Now().UTC(),
			graphmodel.EventKindStart,
		).NextOccurenceAfter(time.Now().UTC()),
		UpdatedAt: server.UpdatedAt,
		CreatedAt: server.CreatedAt,
	}
}

func newModelArchivedServer(server dbmodel.ArchivedServer) *graphmodel.ArchivedServer {
	return &graphmodel.ArchivedServer{
		ID:         server.ID.String(),
		Definition: newModelServerDefinition(server.ServerDefinition),
		UpdatedAt:  server.UpdatedAt,
		CreatedAt:  server.CreatedAt,
	}
}
