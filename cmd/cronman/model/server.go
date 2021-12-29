package model

import "github.com/google/uuid"

type Server struct {
	Model
	Name                   string         `json:"name"`
	InstanceID             string         `json:"instanceID"`
	InstanceKind           InstanceKind   `json:"instanceKind"`
	AllocationID           string         `json:"allocationID"`
	ElasticIP              string         `json:"elasticIP"`
	MaxPlayers             uint16         `json:"maxPlayers"`
	MapSize                uint16         `json:"mapSize"`
	MapSeed                uint16         `json:"mapSeed"`
	MapSalt                uint16         `json:"mapSalt"`
	TickRate               uint8          `json:"tickRate"`
	RconPassword           string         `json:"rconPassword"`
	Description            string         `json:"description"`
	Background             BackgroundKind `json:"background"`
	Url                    string         `json:"url"`
	BannerUrl              string         `json:"bannerURL"`
	WipeDay                WipeDay        `json:"wipeDay"`
	BlueprintWipeFrequency WipeFrequency  `json:"blueprintWipeFrequency"`
	MapWipeFrequency       WipeFrequency  `json:"mapWipeFrequency"`
	Region                 Region         `json:"region"`
	Tags                   Tags           `json:"tags"`
	Events                 Events         `json:"events"`
	Moderators             Moderators     `json:"moderators"`
}

func (s Server) Clone() *Server {
	cloned := s
	cloned.Tags = s.Tags.Clone()
	cloned.Events = s.Events.Clone()
	cloned.Moderators = s.Moderators.Clone()
	return &cloned
}

func (s *Server) Scrub() {
	s.Model.Scrub()
	s.InstanceID = "instance-ID"
	s.AllocationID = "allocation-ID"
	s.ElasticIP = "elastic-IP"

	s.Tags.Scrub()
	s.Events.Scrub()
	s.Moderators.Scrub()
}

type LiveServers []LiveServer

func (s LiveServers) Clone() LiveServers {
	cloned := make(LiveServers, 0, len(s))
	cloned = append(cloned, s...)
	return cloned
}

func (s LiveServers) Scrub() {
	for i := range s {
		s[i].Scrub()
	}
}

type LiveServer struct {
	Model

	ServerID uuid.UUID
	Server

	AssociationID string
	ActivePlayers uint8
	QueuedPlayers uint8
}

func (s LiveServer) Clone() LiveServer {
	return s
}

func (s *LiveServer) Scrub() {
	s.Model.Scrub()
	s.AssociationID = ""
	s.Server.Scrub()
}

type DormantServers []DormantServer

func (s DormantServers) Clone() DormantServers {
	cloned := make(DormantServers, 0, len(s))
	cloned = append(cloned, s...)
	return cloned
}

func (s DormantServers) Scrub() {
	for i := range s {
		s[i].Scrub()
	}
}

type DormantServer struct {
	Model

	ServerID uuid.UUID
	Server
}

func (s DormantServer) Clone() DormantServer {
	return s
}

func (s *DormantServer) Scrub() {
	s.Model.Scrub()
	s.Server.Scrub()
}

type ArchivedServers []ArchivedServer

func (s ArchivedServers) Clone() ArchivedServers {
	cloned := make(ArchivedServers, 0, len(s))
	cloned = append(cloned, s...)
	return cloned
}

func (s ArchivedServers) Scrub() {
	for i := range s {
		s[i].Scrub()
	}
}

type ArchivedServer struct {
	Model

	ServerID uuid.UUID
	Server
}

func (s ArchivedServer) Clone() ArchivedServer {
	return s
}

func (s *ArchivedServer) Scrub() {
	s.Model.Scrub()
	s.Server.Scrub()
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
