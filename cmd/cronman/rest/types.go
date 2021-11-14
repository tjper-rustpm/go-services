package rest

import (
	"time"

	"github.com/tjper/rustcron/cmd/cronman/model"
)

type CreateServerBody struct {
	Name                   string               `json:"name"`
	InstanceKind           model.InstanceKind   `json:"instanceKind"`
	MaxPlayers             int                  `json:"maxPlayers"`
	MapSize                int                  `json:"mapSize"`
	MapSeed                int                  `json:"mapSeed"`
	MapSalt                int                  `json:"mapSalt"`
	TickRate               int                  `json:"tickRate"`
	RconPassword           string               `json:"rconPassword"`
	Description            string               `json:"description"`
	URL                    string               `json:"url"`
	Background             model.BackgroundKind `json:"background"`
	BannerURL              string               `json:"bannerURL"`
	WipeDay                model.WipeDay        `json:"wipeDay"`
	BlueprintWipeFrequency model.WipeFrequency  `json:"blueprintWipeFrequency"`
	MapWipeFrequency       model.WipeFrequency  `json:"mapWipeFrequency"`
	Region                 model.Region         `json:"region"`

	Events []struct {
		Weekday time.Weekday    `json:"weekday"`
		Hour    int             `json:"hour"`
		Kind    model.EventKind `json:"kind"`
	} `json:"events"`

	Moderators []struct {
		SteamID string `json:"steamID"`
	} `json:"moderators"`

	Tags []struct {
		Description string         `json:"description"`
		Icon        model.IconKind `json:"icon"`
		Value       string         `json:"value"`
	} `json:"tags"`
}

func (body CreateServerBody) ToModelServerDefinition() (*model.ServerDefinition, error) {
	sd := &model.ServerDefinition{}
	if err := jsonConversion(body, sd); err != nil {
		return nil, err
	}
	return sd, nil
}
