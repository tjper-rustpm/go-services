package rest

import (
	"time"

	"github.com/google/uuid"
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

	Events []struct {
		Weekday time.Weekday    `json:"weekday"`
		Hour    uint8           `json:"hour"`
		Kind    model.EventKind `json:"kind"`
	} `json:"events"`

	Moderators []struct {
		SteamID string `json:"steamId"`
	} `json:"moderators"`

	Tags []struct {
		Description string         `json:"description"`
		Icon        model.IconKind `json:"icon"`
		Value       string         `json:"value"`
	} `json:"tags"`
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
		Events:       EventsFromModel(server.Events),
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
	Events       []Event              `json:"events"`
}

type DormantServer struct {
	Header
	Server

	StartsAt  time.Time `json:"startsAt"`
	CreatedAt time.Time `json:"createdAt"`
}

func DormantServerFromModel(dormant model.DormantServer) *DormantServer {
	return &DormantServer{
		Header: Header{
			ID:   dormant.ID,
			Kind: "dormant",
		},
		Server: ServerFromModel(dormant.Server),
		StartsAt: dormant.Server.Events.NextOf(
			time.Now().UTC(),
			model.EventKindStart,
		).NextOccurenceAfter(time.Now().UTC()),
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
			ID:   live.ID,
			Kind: "live",
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
			ID:   archived.ID,
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

type Tag struct {
	ID          uuid.UUID      `json:"id"`
	Description string         `json:"description"`
	Icon        model.IconKind `json:"icon"`
	Value       string         `json:"value"`
}

func EventsFromModel(modelEvents model.Events) []Event {
	events := make([]Event, 0, len(modelEvents))
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

type Event struct {
	ID      uuid.UUID       `json:"id"`
	Weekday time.Weekday    `json:"weekday"`
	Hour    uint8           `json:"hour"`
	Kind    model.EventKind `json:"kind"`
}
