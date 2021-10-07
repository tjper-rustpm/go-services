package model

import (
	"database/sql"

	"github.com/google/uuid"
)

type DefinitionModerators []DefinitionModerator

func (dts DefinitionModerators) Clone() DefinitionModerators {
	cloned := make(DefinitionModerators, 0, len(dts))
	for _, dt := range dts {
		cloned = append(cloned, dt.Clone())
	}
	return cloned
}

func (dts DefinitionModerators) Scrub() {
	for i := range dts {
		dts[i].Scrub()
	}
}

type DefinitionModerator struct {
	Model
	SteamID            string
	QueuedDeletionAt   sql.NullTime
	ServerDefinitionID uuid.UUID
}

func (dm DefinitionModerator) Clone() DefinitionModerator {
	return dm
}

func (dm *DefinitionModerator) Scrub() {
	dm.Model.Scrub()
	dm.QueuedDeletionAt = sql.NullTime{}
	dm.ServerDefinitionID = uuid.Nil
}
