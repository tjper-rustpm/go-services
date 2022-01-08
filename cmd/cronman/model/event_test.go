package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventsNextOf(t *testing.T) {
	type expected struct {
		event Event
	}
	tests := map[string]struct {
		dt     time.Time
		kind   EventKind
		events Events
		exp    expected
	}{
		"single": {
			dt:   time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			kind: EventKindStart,
			events: Events{
				event(1, 22, EventKindStart),
			},
			exp: expected{
				event: event(1, 22, EventKindStart),
			},
		},
		"twelve-ordered": {
			dt:   time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			kind: EventKindStart,
			events: Events{
				event(0, 22, EventKindStart),
				event(1, 4, EventKindStop),
				event(1, 22, EventKindStart),
				event(2, 4, EventKindStop),
				event(2, 22, EventKindStart),
				event(3, 4, EventKindStop),
				event(3, 22, EventKindStart),
				event(4, 4, EventKindStop),
				event(4, 22, EventKindStart),
				event(5, 4, EventKindStop),
				event(5, 22, EventKindStart),
				event(6, 4, EventKindStop),
			},
			exp: expected{
				event: event(3, 22, EventKindStart),
			},
		},
		"twelve-unordered": {
			dt:   time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			kind: EventKindStart,
			events: Events{
				event(1, 22, EventKindStart),
				event(2, 22, EventKindStart),
				event(3, 4, EventKindStop),
				event(2, 4, EventKindStop),
				event(3, 22, EventKindStart),
				event(4, 4, EventKindStop),
				event(5, 22, EventKindStart),
				event(1, 4, EventKindStop),
				event(4, 22, EventKindStart),
				event(5, 4, EventKindStop),
				event(0, 22, EventKindStart),
				event(6, 4, EventKindStop),
			},
			exp: expected{
				event: event(3, 22, EventKindStart),
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			event := test.events.NextOf(test.dt, test.kind)
			assert.Equal(t, test.exp.event, event)
		})
	}
}

func TestEventNextOccurenceAfter(t *testing.T) {
	type expected struct {
		next time.Time
	}
	tests := map[string]struct {
		event Event
		after time.Time
		exp   expected
	}{
		"+day,+hour": {
			event: event(4, 22, EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 17, 22, 0, 0, 0, time.UTC),
			},
		},
		"++day,+hour": {
			event: event(6, 22, EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 19, 22, 0, 0, 0, time.UTC),
			},
		},
		"++day,-hour": {
			event: event(6, 16, EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 19, 16, 0, 0, 0, time.UTC),
			},
		},
		"day,+hour": {
			event: event(3, 22, EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 16, 22, 0, 0, 0, time.UTC),
			},
		},
		"-day,+hour": {
			event: event(2, 22, EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 22, 22, 0, 0, 0, time.UTC),
			},
		},
		"--day,+hour": {
			event: event(0, 22, EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 20, 22, 0, 0, 0, time.UTC),
			},
		},
		"--day,-hour": {
			event: event(0, 16, EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 20, 16, 0, 0, 0, time.UTC),
			},
		},
		"day,hour": {
			event: event(3, 19, EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 23, 19, 0, 0, 0, time.UTC),
			},
		},
		"+day,-hour,+minute": {
			event: event(4, 16, EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 30, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 17, 16, 0, 0, 0, time.UTC),
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			next := test.event.NextOccurenceAfter(test.after)
			assert.Equal(t, test.exp.next, next)
		})
	}
}

// --- helpers ---

func event(day time.Weekday, hour uint8, kind EventKind) Event {
	return Event{
		Weekday: day,
		Hour:    hour,
		Kind:    kind,
	}
}
