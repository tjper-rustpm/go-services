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

func (es Events) NextEventAfter(t time.Time, kind EventKind) Event {
	var next Event
	for _, e := range es {
		if e.Kind != kind {
			continue
		}
		if (next == Event{}) {
			next = e
			continue
		}

		potential := e.NextTimeAfter(t)
		current := next.NextTimeAfter(t)

		if potential.Before(current) {
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

func (e Event) NextTime() time.Time {
	return e.NextTimeAfter(time.Now().UTC())
}

func (e Event) NextTimeAfter(t time.Time) time.Time {
	nextEventWeekday := e.Weekday
	if (e.Weekday < t.Weekday()) || (e.Weekday == t.Weekday() && int(e.Hour) <= t.Hour()) {
		nextEventWeekday += 7
	}

	days := time.Duration(nextEventWeekday-t.Weekday()) * 24 * time.Hour
	hours := time.Duration(int(e.Hour)-t.Hour()) * time.Hour

	return t.Add(days + hours).Truncate(time.Hour)
}

type EventKind string

const (
	EventKindStart EventKind = "start"
	EventKindStop  EventKind = "stop"
)
