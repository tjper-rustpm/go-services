package rest

import (
	"errors"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/controller"
	"github.com/tjper/rustcron/cmd/cronman/model"
	imodel "github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
)

type CreateServerBody struct {
	Name         string                 `json:"name" validate:"required"`
	InstanceKind model.InstanceKind     `json:"instanceKind" validate:"required"`
	MaxPlayers   uint16                 `json:"maxPlayers" validate:"required"`
	MapSize      model.MapSizeKind      `json:"mapSize" validate:"required"`
	MapSeed      uint32                 `json:"mapSeed" validate:"required"`
	MapSalt      uint32                 `json:"mapSalt" validate:"required"`
	TickRate     uint8                  `json:"tickRate" validate:"required"`
	RconPassword string                 `json:"rconPassword" validate:"required"`
	Description  string                 `json:"description" validate:"required"`
	URL          string                 `json:"url" validate:"required,url"`
	Background   model.BackgroundKind   `json:"background" validate:"required"`
	BannerURL    string                 `json:"bannerURL" validate:"required,url"`
	Region       model.Region           `json:"region" validate:"required"`
	Options      map[string]interface{} `json:"options"`

	Events     Events     `json:"events" validate:"required,min=1,dive,required"`
	Moderators Moderators `json:"moderators" validate:"required,dive,required"`
	Owners     Owners     `json:"owners" validate:"required,min=1,dive,required"`
	Tags       Tags       `json:"tags" validate:"required,dive,required"`
}

func (body CreateServerBody) ToModelServer(id uuid.UUID) model.Server {
	events := make(model.Events, 0, len(body.Events))
	for _, event := range body.Events {
		events = append(
			events,
			model.Event{Schedule: event.Schedule, Weekday: event.Weekday, Kind: event.Kind},
		)
	}

	moderators := make(model.Moderators, 0, len(body.Moderators))
	for _, moderator := range body.Moderators {
		moderators = append(
			moderators,
			model.Moderator{SteamID: moderator.SteamID},
		)
	}

	owners := make(model.Owners, 0, len(body.Owners))
	for _, owner := range body.Owners {
		owners = append(
			owners,
			model.Owner{SteamID: owner.SteamID},
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
		Model:        imodel.Model{ID: id},
		Name:         body.Name,
		InstanceKind: body.InstanceKind,
		MaxPlayers:   body.MaxPlayers,
		MapSize:      body.MapSize,
		TickRate:     body.TickRate,
		RconPassword: body.RconPassword,
		Description:  body.Description,
		URL:          body.URL,
		Background:   body.Background,
		BannerURL:    body.BannerURL,
		Region:       body.Region,
		Options:      body.Options,
		Wipes: model.Wipes{
			{Kind: model.WipeKindFull, MapSeed: body.MapSeed, MapSalt: body.MapSalt},
		},
		Events:     events,
		Moderators: moderators,
		Owners:     owners,
		Tags:       tags,
	}
}

var errModeratorsAndOwnersCollision = errors.New("moderators and owners for a given server must be unique")

func (body CreateServerBody) validateOwnerAndModeratorIntersection() error {
	intersection := make(map[string]struct{})
	for _, moderator := range body.Moderators {
		intersection[moderator.SteamID] = struct{}{}
	}
	for _, owner := range body.Owners {
		if _, ok := intersection[owner.SteamID]; ok {
			return errModeratorsAndOwnersCollision
		}
	}
	return nil
}

type CreateServerResponse struct {
	ID uuid.UUID `json:"id"`
}

func (r *CreateServerResponse) FromUUID(id uuid.UUID) {
	*r = CreateServerResponse{
		ID: id,
	}
}

type PutServerBody struct {
	ID      uuid.UUID              `json:"id" validate:"required"`
	Changes map[string]interface{} `json:"changes" validate:"required,dive,keys,eq=name|eq=instanceKind|eq=maxPlayers|eq=mapSize|eq=mapSeed|eq=mapSalt|eq=tickRate|eq=rconPassword|eq=description|eq=url|eq=background|eq=bannerURL|eq=wipeDay|eq=blueprintWipeFrequency|eq=mapWipeFrequency|eq=region|eq=events|eq=moderators|eq=tags"`
}

func (body PutServerBody) ToUpdateServerInput() controller.UpdateServerInput {
	return controller.UpdateServerInput{ID: body.ID, Changes: body.Changes}
}

type AddServerTagsBody struct {
	ServerID uuid.UUID `json:"serverId" validate:"required"`
	Tags     Tags      `json:"tags" validate:"required"`
}

type RemoveServerTagsBody struct {
	ServerID uuid.UUID   `json:"serverId" validate:"required"`
	TagIDs   []uuid.UUID `json:"tagIds" validate:"required"`
}

type AddServerEventsBody struct {
	ServerID uuid.UUID `json:"serverId" validate:"required"`
	Events   Events    `json:"events" validate:"required"`
}

type RemoveServerEventsBody struct {
	ServerID uuid.UUID   `json:"serverId" validate:"required"`
	EventIDs []uuid.UUID `json:"eventIds" validate:"required"`
}

type AddServerModeratorsBody struct {
	ServerID   uuid.UUID  `json:"serverId" validate:"required"`
	Moderators Moderators `json:"moderators" validate:"required"`
}

type RemoveServerModeratorsBody struct {
	ServerID     uuid.UUID   `json:"serverId" validate:"required"`
	ModeratorIDs []uuid.UUID `json:"moderatorIds" validate:"required"`
}

type AddServerOwnersBody struct {
	ServerID uuid.UUID `json:"serverId" validate:"required"`
	Owners   Owners    `json:"owners" validate:"required"`
}

type RemoveServerOwnersBody struct {
	ServerID uuid.UUID   `json:"serverId" validate:"required"`
	OwnerIDs []uuid.UUID `json:"ownerIds" validate:"required"`
}

func ServerFromModel(server model.Server) *Server {
	return &Server{
		Name:         server.Name,
		InstanceKind: server.InstanceKind,
		ElasticIP:    server.ElasticIP,
		MaxPlayers:   server.MaxPlayers,
		MapSize:      server.MapSize,
		MapSeed:      server.Wipes.CurrentWipe().MapSeed,
		MapSalt:      server.Wipes.CurrentWipe().MapSalt,
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
	MapSize      model.MapSizeKind    `json:"mapSize"`
	MapSeed      uint32               `json:"mapSeed"`
	MapSalt      uint32               `json:"mapSalt"`
	TickRate     uint8                `json:"tickRate"`
	Description  string               `json:"description"`
	Background   model.BackgroundKind `json:"background"`
	Tags         Tags                 `json:"tags"`
	Events       Events               `json:"events"`
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

func DormantServerFromModel(dormant model.DormantServer) (*DormantServer, error) {
	_, at, err := dormant.Server.Events.NextEvent(time.Now().UTC(), model.EventKindLive)
	if err != nil {
		return nil, err
	}

	server := ServerFromModel(dormant.Server)
	return &DormantServer{
		Header: Header{
			ID:   dormant.Server.ID,
			Kind: dormantKind,
		},
		Server:    *server,
		StartsAt:  *at,
		CreatedAt: dormant.CreatedAt,
	}, err
}

type LiveServer struct {
	Header
	Server

	ActivePlayers uint8     `json:"activePlayers"`
	QueuedPlayers uint8     `json:"queuedPlayers"`
	CreatedAt     time.Time `json:"createdAt"`
}

func LiveServerFromModel(live model.LiveServer) *LiveServer {
	server := ServerFromModel(live.Server)
	return &LiveServer{
		Header: Header{
			ID:   live.Server.ID,
			Kind: liveKind,
		},
		Server:        *server,
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
	server := ServerFromModel(archived.Server)
	return &ArchivedServer{
		Header: Header{
			ID:   archived.Server.ID,
			Kind: "archived",
		},
		Server: *server,
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
	Description string         `json:"description" validate:"required"`
	Icon        model.IconKind `json:"icon" validate:"required"`
	Value       string         `json:"value" validate:"required"`
}

func EventsFromModel(modelEvents model.Events) Events {
	events := make(Events, 0, len(modelEvents))
	for _, event := range modelEvents {
		events = append(
			events,
			Event{
				ID:       event.ID,
				Schedule: event.Schedule,
				Weekday:  event.Weekday,
				Kind:     event.Kind,
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
				Schedule: event.Schedule,
				Weekday:  event.Weekday,
				Kind:     event.Kind,
			},
		)
	}
	return modelEvents
}

type Event struct {
	ID       uuid.UUID       `json:"id"`
	Schedule string          `json:"schedule" validate:"required,cron"`
	Weekday  *time.Weekday   `json:"weekday,omitempty" validate:"omitempty,min=0,max=6"`
	Kind     model.EventKind `json:"kind" validate:"required"`
}

type EventAt struct {
	ID   uuid.UUID       `json:"id"`
	At   time.Time       `json:"at"`
	Kind model.EventKind `json:"kind"`
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
	SteamID string    `json:"steamId" validate:"required"`
}

func OwnersFromModel(modelOwners model.Owners) Owners {
	owners := make(Owners, 0, len(modelOwners))
	for _, owner := range modelOwners {
		owners = append(
			owners,
			Owner{
				ID:      owner.ID,
				SteamID: owner.SteamID,
			},
		)
	}
	return owners
}

type Owners []Owner

func (mods Owners) ToModelOwners() model.Owners {
	modelOwners := make(model.Owners, 0, len(mods))
	for _, owner := range mods {
		modelOwners = append(
			modelOwners,
			model.Owner{
				SteamID: owner.SteamID,
			},
		)
	}
	return modelOwners
}

type Owner struct {
	ID      uuid.UUID `json:"id"`
	SteamID string    `json:"steamId" validate:"required"`
}
