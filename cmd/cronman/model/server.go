package model

import (
	"context"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/userdata"
	"github.com/tjper/rustcron/internal/model"
	"gorm.io/gorm"

	"github.com/google/uuid"
)

const (
	LiveServerState    = "servers.live_servers"
	DormantServerState = "servers.dormant_servers"
)

// Server is cronman server and contains general server state. Other server
// types LiveServer, DormanServer, etc are composed of this type.
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
	Vips       Vips
}

// Create creates a Server in the specified db. Non empty relationships will
// be created as well (Wipes, Tags, Events, Moderators, etc).
func (s *Server) Create(ctx context.Context, db *gorm.DB) error {
	if err := db.WithContext(ctx).Create(s).Error; err != nil {
		return fmt.Errorf("model Server.Create: %w", err)
	}
	return nil
}

// First retrieves the server from the specified db by its ID. If the server is
// not found internal/gorm.ErrNotFound is returned.
func (s *Server) First(ctx context.Context, db *gorm.DB) error {
	if err := db.WithContext(ctx).First(s).Error; err != nil {
		return fmt.Errorf("model Server.First: %w", err)
	}
	return nil
}

func (s *Server) ActiveVips() Vips {
	var vips Vips
	for _, vip := range vips {
		if !time.Now().Before(vip.ExpiresAt) {
			continue
		}
		vips = append(vips, vip)
	}
	return vips
}

// IsLive reports if the server is currently live based on data in-memory.
func (s Server) IsLive() bool {
	return s.StateType == LiveServerState
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

// Find retrieves all LiveServers from the passed gorm.DB. Find implements the
// gorm.Finder interface.
func (s *LiveServers) Find(ctx context.Context, db *gorm.DB) error {
	if res := db.
		WithContext(ctx).
		Preload("Server").
		Preload("Server.Wipes").
		Preload("Server.Tags").
		Preload("Server.Events").
		Preload("Server.Moderators").
		Order("created_at DESC").
		Find(s); res.Error != nil {
		return res.Error
	}
	return nil
}

// LiveServer is a server that users can currently connect to, and is scheduled
// to become dormant at some point in the future.
type LiveServer struct {
	model.Model

	Server Server `json:"server" gorm:"polymorphic:State"`

	AssociationID string `json:"-"`
	ActivePlayers uint8  `json:"activePlayers"`
	QueuedPlayers uint8  `json:"queuedPlayers"`
}

// Create creates the LiveServer in the specified db. Non empty relationships
// will be creates as well (Server).
func (ls *LiveServer) Create(ctx context.Context, db *gorm.DB) error {
	if err := db.WithContext(ctx).Create(ls).Error; err != nil {
		return fmt.Errorf("model LiveServer.Create: %w", err)
	}
	return nil
}

// Update updates the LiveServer in the specified db.
func (ls *LiveServer) Update(ctx context.Context, db *gorm.DB, changes interface{}) error {
	if err := db.WithContext(ctx).Model(ls).Updates(changes).Error; err != nil {
		return fmt.Errorf("while updating live server: %w", err)
	}
	return nil
}

func (ls LiveServer) Clone() LiveServer {
	return ls
}

func (ls *LiveServer) Scrub() {
	ls.Model.Scrub()
	ls.AssociationID = ""
	ls.Server.Scrub()
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

// DormantServer is a server that users cannot currently connect to, and is
// scheduled to be live at some point in the future.
type DormantServer struct {
	model.Model

	Server Server `json:"server" gorm:"polymorphic:State"`
}

// Create creates a DormantServer in the specified db. Non empty relationships
// will be creates as well (Server).
func (ds *DormantServer) Create(ctx context.Context, db *gorm.DB) error {
	if err := db.WithContext(ctx).Create(ds).Error; err != nil {
		return fmt.Errorf("model DormantServer.Create: %w", err)
	}
	return nil
}

func (ds DormantServer) Clone() DormantServer {
	return ds
}

func (ds *DormantServer) Scrub() {
	ds.Model.Scrub()
	ds.Server.Scrub()
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
	InstanceKindSmall    InstanceKind = "small"
	InstanceKindStandard InstanceKind = "standard"
	InstanceKindLarge    InstanceKind = "large"
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
