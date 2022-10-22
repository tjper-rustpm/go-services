package model

import (
	"github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
)

type Owners []Owner

func (dts Owners) Clone() Owners {
	cloned := make(Owners, 0, len(dts))
	for _, dt := range dts {
		cloned = append(cloned, dt.Clone())
	}
	return cloned
}

func (dts Owners) Scrub() {
	for i := range dts {
		dts[i].Scrub()
	}
}

func (dts Owners) SteamIDs() []string {
	steamIDs := make([]string, 0, len(dts))
	for _, mod := range dts {
		steamIDs = append(steamIDs, mod.SteamID)
	}
	return steamIDs
}

type Owner struct {
	model.Model
	SteamID  string
	ServerID uuid.UUID
}

func (dm Owner) Clone() Owner {
	return dm
}

func (dm *Owner) Scrub() {
	dm.Model.Scrub()
	dm.ServerID = uuid.Nil
}
