package model

import (
	"github.com/google/uuid"
)

type Moderators []Moderator

func (dts Moderators) Clone() Moderators {
	cloned := make(Moderators, 0, len(dts))
	for _, dt := range dts {
		cloned = append(cloned, dt.Clone())
	}
	return cloned
}

func (dts Moderators) Scrub() {
	for i := range dts {
		dts[i].Scrub()
	}
}

func (dts Moderators) SteamIDs() []string {
	steamIDs := make([]string, 0, len(dts))
	for _, mod := range dts {
		steamIDs = append(steamIDs, mod.SteamID)
	}
	return steamIDs
}

type Moderator struct {
	Model
	SteamID  string
	ServerID uuid.UUID
}

func (dm Moderator) Clone() Moderator {
	return dm
}

func (dm *Moderator) Scrub() {
	dm.Model.Scrub()
	dm.ServerID = uuid.Nil
}
