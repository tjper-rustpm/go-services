package model

import (
	graphmodel "github.com/tjper/rustcron/cmd/cronman/graph/model"

	"github.com/google/uuid"
)

type DefinitionTags []DefinitionTag

func (dts DefinitionTags) Clone() DefinitionTags {
	cloned := make(DefinitionTags, 0, len(dts))
	for _, dt := range dts {
		cloned = append(cloned, dt.Clone())
	}
	return cloned
}

func (dts DefinitionTags) Scrub() {
	for i := range dts {
		dts[i].Scrub()
	}
}

type DefinitionTag struct {
	Model
	Description        string
	Icon               graphmodel.IconKind
	Value              string
	ServerDefinitionID uuid.UUID
}

func (dt DefinitionTag) Clone() DefinitionTag {
	return dt
}

func (dt *DefinitionTag) Scrub() {
	dt.Model.Scrub()
	dt.ServerDefinitionID = uuid.Nil
}
