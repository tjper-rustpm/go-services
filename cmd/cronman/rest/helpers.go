package rest

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/cronman/model"
)

func toModelEvents(
	serverDefinitionID uuid.UUID,
	events []*graphmodel.NewEvent,
) []model.DefinitionEvent {
	dbEvents := make([]model.DefinitionEvent, 0, len(events))
	for _, event := range events {
		dbEvents = append(dbEvents, model.DefinitionEvent{
			ServerDefinitionID: serverDefinitionID,
			Weekday:            time.Weekday(event.Day),
			Hour:               uint8(event.Hour),
			EventKind:          event.Kind,
		})
	}
	return dbEvents
}

func toModelTags(
	serverDefinitionID uuid.UUID,
	tags []*graphmodel.NewTag,
) []model.DefinitionTag {
	dbTags := make([]model.DefinitionTag, 0, len(tags))
	for _, tag := range tags {
		dbTags = append(dbTags, model.DefinitionTag{
			ServerDefinitionID: serverDefinitionID,
			Description:        tag.Description,
			Icon:               tag.Icon,
			Value:              tag.Value,
		})
	}
	return dbTags
}

func toModelDefinitionModerators(
	serverID uuid.UUID,
	steamIDs []string,
) []model.DefinitionModerator {
	dbModerators := make([]model.DefinitionModerator, 0, len(steamIDs))
	for _, steamID := range steamIDs {
		dbModerators = append(dbModerators, model.DefinitionModerator{
			ServerDefinitionID: serverID,
			SteamID:            steamID,
		})
	}
	return dbModerators
}

func jsonConversion(from, to interface{}) error {
	b, err := json.Marshal(from)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, to); err != nil {
		return err
	}
	return nil
}
