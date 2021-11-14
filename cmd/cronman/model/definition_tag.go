package model

import (
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
	Description        string   `json:"description"`
	Icon               IconKind `json:"icon"`
	Value              string   `json:"value"`
	ServerDefinitionID uuid.UUID
}

func (dt DefinitionTag) Clone() DefinitionTag {
	return dt
}

func (dt *DefinitionTag) Scrub() {
	dt.Model.Scrub()
	dt.ServerDefinitionID = uuid.Nil
}

type IconKind string

const (
	IconKindUserGroup     IconKind = "userGroup"
	IconKindMap           IconKind = "map"
	IconKindGlobe         IconKind = "globe"
	IconKindCalendarDay   IconKind = "calendarDay"
	IconKindCalendarWeek  IconKind = "calendarWeek"
	IconKindCalendarEvent IconKind = "calendarEvent"
	IconKindGames         IconKind = "games"
)
