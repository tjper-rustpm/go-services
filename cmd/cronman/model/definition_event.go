package model

import (
	"time"

	"github.com/google/uuid"
)

type DefinitionEvents []DefinitionEvent

func (dts DefinitionEvents) Clone() DefinitionEvents {
	cloned := make(DefinitionEvents, 0, len(dts))
	for _, dt := range dts {
		cloned = append(cloned, dt.Clone())
	}
	return cloned
}

func (dts DefinitionEvents) Scrub() {
	for i := range dts {
		dts[i].Scrub()
	}
}

func (dts DefinitionEvents) NextOf(t time.Time, kind EventKind) DefinitionEvent {
	var next DefinitionEvent
	for _, dt := range dts {
		if dt.EventKind != kind {
			continue
		}
		if (next == DefinitionEvent{}) {
			next = dt
		}

		futureTime := dt.NextOccurenceAfter(t)
		nextTime := next.NextOccurenceAfter(t)
		if futureTime.Before(nextTime) {
			next = dt
		}
	}
	return next
}

type DefinitionEvent struct {
	Model
	Weekday            time.Weekday `json:"weekday"`
	Hour               uint8        `json:"hour"`
	EventKind          EventKind    `json:"kind"`
	ServerDefinitionID uuid.UUID
}

func (de DefinitionEvent) Clone() DefinitionEvent {
	return de
}

func (de *DefinitionEvent) Scrub() {
	de.Model.Scrub()
	de.ServerDefinitionID = uuid.Nil
}

func (de DefinitionEvent) NextOccurenceAfter(after time.Time) time.Time {
	nextEventWeekday := de.Weekday
	if (de.Weekday < after.Weekday()) || (de.Weekday == after.Weekday() && int(de.Hour) <= after.Hour()) {
		nextEventWeekday += 7
	}
	return after.Add(
		time.Duration(nextEventWeekday-after.Weekday())*24*time.Hour +
			time.Duration(int(de.Hour)-after.Hour())*time.Hour,
	).Truncate(time.Hour)
}

type EventKind string

const (
	EventKindStart EventKind = "START"
	EventKindStop  EventKind = "STOP"
)
