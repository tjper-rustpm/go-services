package model

import (
	"github.com/tjper/rustcron/internal/model"

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
	model.Model
	Description string
	Icon        IconKind
	Value       string
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
	IconKindFingerPrint   IconKind = "fingerPrint"
	IconKindClock         IconKind = "clock"
)
