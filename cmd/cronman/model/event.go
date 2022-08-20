package model

import (
	"fmt"
	"time"

	"github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type Events []Event

// NextEvent retrieves the next Event to occur after t of type kind. The first
// return value is the next Event instance. The second return value is the
// next time at which the event occurs. The third return value is a non-nil
// error if a problem occurred determining the next event.
func (es Events) NextEvent(t time.Time, kind EventKind) (*Event, *time.Time, error) {
	var next Event
	for _, e := range es {
		if e.Kind != kind {
			continue
		}
		if (next == Event{}) {
			next = e
			continue
		}

		potential, err := e.Next(t)
		if err != nil {
			return nil, nil, err
		}
		current, err := next.Next(t)
		if err != nil {
			return nil, nil, err
		}

		if potential.Before(current) {
			next = e
		}
	}

	at, err := next.Next(t)

	return &next, &at, err
}

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

type Event struct {
	model.Model
	Schedule string
	Weekday  *time.Weekday
	Kind     EventKind
	ServerID uuid.UUID
}

func (e Event) Next(after time.Time) (time.Time, error) {
	schedule, err := cron.ParseStandard(e.Schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse schedule; id: %s, error: %w", e.ID, err)
	}

	if e.Weekday == nil {
		return schedule.Next(after), nil
	}

	potential := schedule.Next(after)
	if potential.Weekday() == *e.Weekday {
		return potential, nil
	}
	return e.Next(potential)
}

func (e Event) Occurrences(after, until time.Time) ([]time.Time, error) {
	occurrences := make([]time.Time, 0)
	for {
		next, err := e.Next(after)
		if err != nil {
			return nil, fmt.Errorf("occurrences; id: %s, error: %w", e.ID, err)
		}

		if next.After(until) {
			return occurrences, nil
		}

		occurrences = append(occurrences, next)
		after = next
	}
}

func (e Event) IsWeekDay(t time.Time) bool {
	if e.Weekday == nil {
		return false
	}
	return t.Weekday() == *e.Weekday
}

func (e Event) Clone() Event {
	return e
}

func (e *Event) Scrub() {
	e.Model.Scrub()
	e.ServerID = uuid.Nil
}

type EventKind string

const (
	EventKindStart    EventKind = "start"
	EventKindStop     EventKind = "stop"
	EventKindLive     EventKind = "live"
	EventKindFullWipe EventKind = "fullWipe"
	EventKindMapWipe  EventKind = "mapWipe"
)
