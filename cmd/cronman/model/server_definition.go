package model

// ServerDefinition is general server information shared by all server types.
type ServerDefinition struct {
	Model
	Name                   string               `json:"name"`
	InstanceID             string               `json:"instanceID"`
	InstanceKind           InstanceKind         `json:"instanceKind"`
	AllocationID           string               `json:"maxPlayers"`
	ElasticIP              string               `json:"elasticIP"`
	MaxPlayers             uint16               `json:"maxPlayers"`
	MapSize                uint16               `json:"mapSize"`
	MapSeed                uint16               `json:"mapSeed"`
	MapSalt                uint16               `json:"mapSalt"`
	TickRate               uint8                `json:"tickRate"`
	RconPassword           string               `json:"rconPassword"`
	Description            string               `json:"description"`
	Background             BackgroundKind       `json:"background"`
	Url                    string               `json:"url"`
	BannerUrl              string               `json:"bannerURL"`
	WipeDay                WipeDay              `json:"wipeDay"`
	BlueprintWipeFrequency WipeFrequency        `json:"blueprintWipeFrequency"`
	MapWipeFrequency       WipeFrequency        `json:"mapWipeFrequency"`
	Region                 Region               `json:"region"`
	Tags                   DefinitionTags       `json:"tags"`
	Events                 DefinitionEvents     `json:"events"`
	Moderators             DefinitionModerators `json:"moderators"`
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
