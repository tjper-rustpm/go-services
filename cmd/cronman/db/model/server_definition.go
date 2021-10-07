package model

import graphmodel "github.com/tjper/rustcron/cmd/cronman/graph/model"

// ServerDefinition is general server information shared by all server types.
type ServerDefinition struct {
	Model
	Name                   string
	InstanceID             string
	InstanceKind           graphmodel.InstanceKind
	AllocationID           string
	ElasticIP              string
	MaxPlayers             uint16
	MapSize                uint16
	MapSeed                uint16
	MapSalt                uint16
	TickRate               uint8
	RconPassword           string
	Description            string
	Background             graphmodel.BackgroundKind
	Url                    string
	BannerUrl              string
	WipeDay                graphmodel.WipeDay
	BlueprintWipeFrequency graphmodel.WipeFrequency
	MapWipeFrequency       graphmodel.WipeFrequency
	Region                 graphmodel.Region
	Tags                   DefinitionTags
	Events                 DefinitionEvents
	Moderators             DefinitionModerators
}

func (sd ServerDefinition) Clone() *ServerDefinition {
	cloned := sd
	cloned.Tags = sd.Tags.Clone()
	cloned.Events = sd.Events.Clone()
	cloned.Moderators = sd.Moderators.Clone()
	return &cloned
}

func (sd *ServerDefinition) Scrub() {
	sd.Model.Scrub()
	sd.InstanceID = "instance-ID"
	sd.AllocationID = "allocation-ID"
	sd.ElasticIP = "elastic-IP"

	sd.Tags.Scrub()
	sd.Events.Scrub()
	sd.Moderators.Scrub()
}

type InstanceKind string

const (
	InstanceKindStandard InstanceKind = "standard"
)

type BackgroundKind string

const (
	BackgroundKindAirport          BackgroundKind = "airport"
	BackgroundKindBeachLighthouse  BackgroundKind = "beachLighthouse"
	BackgroundKindBigOilNight      BackgroundKind = "bigOilNight"
	BackgroundKindForest           BackgroundKind = "forest"
	BackgroundKindIslandLighthouse BackgroundKind = "islandLighthouse"
	BackgroundKindJunkyard         BackgroundKind = "junkyard"
	BackgroundKindMountainNight    BackgroundKind = "mountainNight"
	BackgroundKindOxum             BackgroundKind = "oxum"
	BackgroundKindSewerNight       BackgroundKind = "sewerNight"
	BackgroundKindTowerNight       BackgroundKind = "towerNight"
)

type WipeDay string

const (
	WipeDaySunday    WipeDay = "sunday"
	WipeDayMonday    WipeDay = "monday"
	WipeDayTuesday   WipeDay = "tuesday"
	WipeDayWednesday WipeDay = "wednesday"
	WipeDayThursday  WipeDay = "thursday"
	WipeDayFriday    WipeDay = "friday"
	WipeDaySaturday  WipeDay = "saturday"
)

type WipeFrequency string

const (
	WipeFrequencyWeekly   WipeFrequency = "weekly"
	WipeFrequencyBiWeekly WipeFrequency = "biweekly"
	WipeFrequencyMonthly  WipeFrequency = "monthly"
)

type Region string

const (
	RegionUsEast    Region = "usEast"
	RegionUsWest    Region = "usWest"
	RegionEuCentral Region = "euCentral"
)
