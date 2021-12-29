package model

import (
	"github.com/google/uuid"
)

type Tags []Tag

func (ts Tags) Clone() Tags {
	cloned := make(Tags, 0, len(ts))
	for _, t := range ts {
		cloned = append(cloned, t.Clone())
	}
	return cloned
}

func (ts Tags) Scrub() {
	for i := range ts {
		ts[i].Scrub()
	}
}

type Tag struct {
	Model
	Description string   `json:"description"`
	Icon        IconKind `json:"icon"`
	Value       string   `json:"value"`
	ServerID    uuid.UUID
}

func (t Tag) Clone() Tag { return t }

func (t *Tag) Scrub() {
	t.Model.Scrub()
	t.ServerID = uuid.Nil
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
