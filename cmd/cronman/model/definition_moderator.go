package model

import (
	"database/sql"

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

type Moderator struct {
	Model
	SteamID          string `json:"steamID"`
	QueuedDeletionAt sql.NullTime
	ServerID         uuid.UUID
}

func (dm Moderator) Clone() Moderator {
	return dm
}

func (dm *Moderator) Scrub() {
	dm.Model.Scrub()
	dm.QueuedDeletionAt = sql.NullTime{}
	dm.ServerID = uuid.Nil
}
