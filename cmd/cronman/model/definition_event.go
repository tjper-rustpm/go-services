package model

import (
	"time"

	"github.com/google/uuid"
)

type Events []Event

func (es Events) Clone() Events {
	cloned := make(Events, 0, len(es))
	for _, e := range es {
		cloned = append(cloned, e.Clone())
	}
	return cloned
}

func (es Events) Scrub() {
	for i := range es {
		es[i].Scrub()
	}
}

func (es Events) NextOf(t time.Time, kind EventKind) Event {
	var next Event
	for _, e := range es {
		if e.Kind != kind {
			continue
		}
		if (next == Event{}) {
			next = e
		}

		futureTime := e.NextOccurenceAfter(t)
		nextTime := next.NextOccurenceAfter(t)
		if futureTime.Before(nextTime) {
			next = e
		}
	}
	return next
}

type Event struct {
	Model
	Weekday  time.Weekday
	Hour     uint8
	Kind     EventKind
	ServerID uuid.UUID
}

func (e Event) Clone() Event {
	return e
}

func (e *Event) Scrub() {
	e.Model.Scrub()
	e.ServerID = uuid.Nil
}

func (e Event) NextOccurenceAfter(after time.Time) time.Time {
	nextEventWeekday := e.Weekday
	if (e.Weekday < after.Weekday()) || (e.Weekday == after.Weekday() && int(e.Hour) <= after.Hour()) {
		nextEventWeekday += 7
	}
	return after.Add(
		time.Duration(nextEventWeekday-after.Weekday())*24*time.Hour +
			time.Duration(int(e.Hour)-after.Hour())*time.Hour,
	).Truncate(time.Hour)
}

type EventKind string

const (
	EventKindStart EventKind = "START"
	EventKindStop  EventKind = "STOP"
)
