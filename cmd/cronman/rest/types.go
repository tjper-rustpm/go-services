package rest

import (
	"time"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/cronman/controller"
	"github.com/tjper/rustcron/cmd/cronman/model"
)

type CreateServerBody struct {
	Name                   string               `json:"name"`
	InstanceKind           model.InstanceKind   `json:"instanceKind"`
	MaxPlayers             uint16               `json:"maxPlayers"`
	MapSize                uint16               `json:"mapSize"`
	MapSeed                uint16               `json:"mapSeed"`
	MapSalt                uint16               `json:"mapSalt"`
	TickRate               uint8                `json:"tickRate"`
	RconPassword           string               `json:"rconPassword"`
	Description            string               `json:"description"`
	Url                    string               `json:"url"`
	Background             model.BackgroundKind `json:"background"`
	BannerUrl              string               `json:"bannerUrl"`
	WipeDay                model.WipeDay        `json:"wipeDay"`
	BlueprintWipeFrequency model.WipeFrequency  `json:"blueprintWipeFrequency"`
	MapWipeFrequency       model.WipeFrequency  `json:"mapWipeFrequency"`
	Region                 model.Region         `json:"region"`

	Events Events `json:"events"`

	Moderators []struct {
		SteamID string `json:"steamId"`
	} `json:"moderators"`

	Tags Tags `json:"tags"`
}

func (body CreateServerBody) ToModelServer() model.Server {
	events := make(model.Events, 0, len(body.Events))
	for _, event := range body.Events {
		events = append(
			events,
			model.Event{Weekday: event.Weekday, Hour: event.Hour, Kind: event.Kind},
		)
	}

	moderators := make(model.Moderators, 0, len(body.Moderators))
	for _, moderator := range body.Moderators {
		moderators = append(
			moderators,
			model.Moderator{SteamID: moderator.SteamID},
		)
	}

	tags := make([]model.Tag, 0, len(body.Tags))
	for _, tag := range body.Tags {
		tags = append(
			tags,
			model.Tag{Description: tag.Description, Icon: tag.Icon, Value: tag.Value},
		)
	}

	return model.Server{
		Name:                   body.Name,
		InstanceKind:           body.InstanceKind,
		MaxPlayers:             body.MaxPlayers,
		MapSize:                body.MapSize,
		MapSeed:                body.MapSeed,
		MapSalt:                body.MapSalt,
		TickRate:               body.TickRate,
		RconPassword:           body.RconPassword,
		Description:            body.Description,
		Url:                    body.Url,
		Background:             body.Background,
		BannerUrl:              body.BannerUrl,
		WipeDay:                body.WipeDay,
		BlueprintWipeFrequency: body.BlueprintWipeFrequency,
		MapWipeFrequency:       body.MapWipeFrequency,
		Region:                 body.Region,
		Events:                 events,
		Moderators:             moderators,
		Tags:                   tags,
	}
}

type PutServerBody struct {
	ID      uuid.UUID              `json:"id"`
	Changes map[string]interface{} `json:"changes"`
}

func (body PutServerBody) ToUpdateServerInput() controller.UpdateServerInput {
	return controller.UpdateServerInput{ID: body.ID, Changes: body.Changes}
}

type AddServerTagsBody struct {
	ServerID uuid.UUID `json:"serverId"`
	Tags     Tags      `json:"tags"`
}

type RemoveServerTagsBody struct {
	ServerID uuid.UUID   `json:"serverId"`
	TagIDs   []uuid.UUID `json:"tagIds"`
}

type AddServerEventsBody struct {
	ServerID uuid.UUID `json:"serverId"`
	Events   Events    `json:"events"`
}

type RemoveServerEventsBody struct {
	ServerID uuid.UUID   `json:"serverId"`
	EventIDs []uuid.UUID `json:"eventIds"`
}

type AddServerModeratorsBody struct {
	ServerID   uuid.UUID  `json:"serverId"`
	Moderators Moderators `json:"moderators"`
}

type RemoveServerModeratorsBody struct {
	ServerID     uuid.UUID   `json:"serverId"`
	ModeratorIDs []uuid.UUID `json:"moderatorIds"`
}

func ServerFromModel(server model.Server) Server {
	return Server{
		Name:         server.Name,
		InstanceKind: server.InstanceKind,
		ElasticIP:    server.ElasticIP,
		MaxPlayers:   server.MaxPlayers,
		MapSize:      server.MapSize,
		MapSeed:      server.MapSeed,
		MapSalt:      server.MapSalt,
		TickRate:     server.TickRate,
		Description:  server.Description,
		Background:   server.Background,
		Tags:         TagsFromModel(server.Tags),
		Events:       EventsAtFromModel(server.Events),
	}
}

type Server struct {
	Name         string               `json:"name"`
	InstanceKind model.InstanceKind   `json:"instanceKind"`
	ElasticIP    string               `json:"elasticIP"`
	MaxPlayers   uint16               `json:"maxPlayers"`
	MapSize      uint16               `json:"mapSize"`
	MapSeed      uint16               `json:"mapSeed"`
	MapSalt      uint16               `json:"mapSalt"`
	TickRate     uint8                `json:"tickRate"`
	Description  string               `json:"description"`
	Background   model.BackgroundKind `json:"background"`
	Tags         []Tag                `json:"tags"`
	Events       []EventAt            `json:"events"`
}

const (
	dormantKind = "dormant"
	liveKind    = "live"
)

type DormantServer struct {
	Header
	Server

	StartsAt  time.Time `json:"startsAt"`
	CreatedAt time.Time `json:"createdAt"`
}

func DormantServerFromModel(dormant model.DormantServer) *DormantServer {
	return &DormantServer{
		Header: Header{
			ID:   dormant.Server.ID,
			Kind: dormantKind,
		},
		Server: ServerFromModel(dormant.Server),
		StartsAt: dormant.Server.Events.NextEventAfter(
			time.Now().UTC(),
			model.EventKindStart,
		).NextTime(),
		CreatedAt: dormant.CreatedAt,
	}
}

type LiveServer struct {
	Header
	Server

	ActivePlayers uint8     `json:"activePlayers"`
	QueuedPlayers uint8     `json:"queuedPlayers"`
	CreatedAt     time.Time `json:"createdAt"`
}

func LiveServerFromModel(live model.LiveServer) *LiveServer {
	return &LiveServer{
		Header: Header{
			ID:   live.Server.ID,
			Kind: liveKind,
		},
		Server:        ServerFromModel(live.Server),
		ActivePlayers: live.ActivePlayers,
		QueuedPlayers: live.QueuedPlayers,
		CreatedAt:     live.CreatedAt,
	}
}

type ArchivedServer struct {
	Header
	Server
}

func ArchivedServerFromModel(archived model.ArchivedServer) *ArchivedServer {
	return &ArchivedServer{
		Header: Header{
			ID:   archived.Server.ID,
			Kind: "archived",
		},
		Server: ServerFromModel(archived.Server),
	}
}

type Header struct {
	ID   uuid.UUID `json:"id"`
	Kind string    `json:"kind"`
}

func TagsFromModel(modelTags model.Tags) []Tag {
	tags := make([]Tag, 0, len(modelTags))
	for _, tag := range modelTags {
		tags = append(
			tags,
			Tag{
				ID:          tag.ID,
				Description: tag.Description,
				Icon:        tag.Icon,
				Value:       tag.Value,
			},
		)
	}
	return tags
}

type Tags []Tag

func (tags Tags) ToModelTags() model.Tags {
	modelTags := make(model.Tags, 0, len(tags))
	for _, tag := range tags {
		modelTags = append(
			modelTags,
			model.Tag{
				Description: tag.Description,
				Icon:        tag.Icon,
				Value:       tag.Value,
			},
		)
	}
	return modelTags
}

type Tag struct {
	ID          uuid.UUID      `json:"id"`
	Description string         `json:"description"`
	Icon        model.IconKind `json:"icon"`
	Value       string         `json:"value"`
}

func EventsAtFromModel(modelEvents model.Events) []EventAt {
	events := make([]EventAt, 0, len(modelEvents))
	for _, event := range modelEvents {
		events = append(
			events,
			EventAt{
				ID:   event.ID,
				At:   event.NextTime(),
				Kind: event.Kind,
			},
		)
	}
	return events
}

type EventAt struct {
	ID   uuid.UUID       `json:"id"`
	Kind model.EventKind `json:"kind"`
	At   time.Time       `json:"at"`
}

func EventsFromModel(modelEvents model.Events) Events {
	events := make(Events, 0, len(modelEvents))
	for _, event := range modelEvents {
		events = append(
			events,
			Event{
				ID:      event.ID,
				Weekday: event.Weekday,
				Hour:    event.Hour,
				Kind:    event.Kind,
			},
		)
	}
	return events
}

type Events []Event

func (events Events) ToModelEvents() model.Events {
	modelEvents := make(model.Events, 0, len(events))
	for _, event := range events {
		modelEvents = append(
			modelEvents,
			model.Event{
				Weekday: event.Weekday,
				Hour:    event.Hour,
				Kind:    event.Kind,
			},
		)
	}
	return modelEvents
}

type Event struct {
	ID      uuid.UUID       `json:"id"`
	Weekday time.Weekday    `json:"weekday"`
	Hour    uint8           `json:"hour"`
	Kind    model.EventKind `json:"kind"`
}

func ModeratorsFromModel(modelModerators model.Moderators) Moderators {
	moderators := make(Moderators, 0, len(modelModerators))
	for _, moderator := range modelModerators {
		moderators = append(
			moderators,
			Moderator{
				ID:      moderator.ID,
				SteamID: moderator.SteamID,
			},
		)
	}
	return moderators
}

type Moderators []Moderator

func (mods Moderators) ToModelModerators() model.Moderators {
	modelModerators := make(model.Moderators, 0, len(mods))
	for _, moderator := range mods {
		modelModerators = append(
			modelModerators,
			model.Moderator{
				SteamID: moderator.SteamID,
			},
		)
	}
	return modelModerators
}

type Moderator struct {
	ID      uuid.UUID `json:"id"`
	SteamID string    `json:"steamId"`
}
