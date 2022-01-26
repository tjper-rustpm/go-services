package model

import (
	"github.com/tjper/rustcron/cmd/cronman/userdata"
	"github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
)

const (
	LiveServerState    = "rustpm.live_servers"
	DormantServerState = "rustpm.dormant_servers"
)

type Server struct {
	model.Model

	StateID   uuid.UUID
	StateType string

	Name         string
	InstanceID   string
	InstanceKind InstanceKind
	AllocationID string
	ElasticIP    string
	MaxPlayers   uint16
	MapSize      uint16
	TickRate     uint8
	RconPassword string
	Description  string
	Background   BackgroundKind
	URL          string
	BannerURL    string
	Region       Region

	Wipes      Wipes
	Tags       Tags
	Events     Events
	Moderators Moderators
}

// UserData generates the userdata to be used by AWS to launch the server in
// proper state.
func (s Server) Userdata(options ...userdata.Option) string {
	return userdata.Generate(
		s.Name,
		s.RconPassword,
		int(s.MaxPlayers),
		int(s.MapSize),
		int(s.Wipes.CurrentWipe().MapSeed),
		int(s.Wipes.CurrentWipe().MapSalt),
		int(s.TickRate),
		options...,
	)
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
	model.Model

	Server Server `json:"server" gorm:"polymorphic:State"`

	AssociationID string `json:"-"`
	ActivePlayers uint8  `json:"activePlayers"`
	QueuedPlayers uint8  `json:"queuedPlayers"`
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
	model.Model

	Server Server `json:"server" gorm:"polymorphic:State"`
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
	model.Model

	Server Server `json:"server" gorm:"polymorphic:State"`
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

type Region string

const (
	RegionUsEast    Region = "usEast"
	RegionUsWest    Region = "usWest"
	RegionEuCentral Region = "euCentral"
)
