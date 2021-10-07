package model

import (
	"testing"
	"time"

	graphmodel "github.com/tjper/rustcron/cmd/cronman/graph/model"

	"github.com/stretchr/testify/assert"
)

func TestDefinitionEventsNextOf(t *testing.T) {
	type expected struct {
		event DefinitionEvent
	}
	tests := map[string]struct {
		dt     time.Time
		kind   graphmodel.EventKind
		events DefinitionEvents
		exp    expected
	}{
		"single": {
			dt:   time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			kind: graphmodel.EventKindStart,
			events: DefinitionEvents{
				event(1, 22, graphmodel.EventKindStart),
			},
			exp: expected{
				event: event(1, 22, graphmodel.EventKindStart),
			},
		},
		"twelve-ordered": {
			dt:   time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			kind: graphmodel.EventKindStart,
			events: DefinitionEvents{
				event(0, 22, graphmodel.EventKindStart),
				event(1, 4, graphmodel.EventKindStop),
				event(1, 22, graphmodel.EventKindStart),
				event(2, 4, graphmodel.EventKindStop),
				event(2, 22, graphmodel.EventKindStart),
				event(3, 4, graphmodel.EventKindStop),
				event(3, 22, graphmodel.EventKindStart),
				event(4, 4, graphmodel.EventKindStop),
				event(4, 22, graphmodel.EventKindStart),
				event(5, 4, graphmodel.EventKindStop),
				event(5, 22, graphmodel.EventKindStart),
				event(6, 4, graphmodel.EventKindStop),
			},
			exp: expected{
				event: event(3, 22, graphmodel.EventKindStart),
			},
		},
		"twelve-unordered": {
			dt:   time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			kind: graphmodel.EventKindStart,
			events: DefinitionEvents{
				event(1, 22, graphmodel.EventKindStart),
				event(2, 22, graphmodel.EventKindStart),
				event(3, 4, graphmodel.EventKindStop),
				event(2, 4, graphmodel.EventKindStop),
				event(3, 22, graphmodel.EventKindStart),
				event(4, 4, graphmodel.EventKindStop),
				event(5, 22, graphmodel.EventKindStart),
				event(1, 4, graphmodel.EventKindStop),
				event(4, 22, graphmodel.EventKindStart),
				event(5, 4, graphmodel.EventKindStop),
				event(0, 22, graphmodel.EventKindStart),
				event(6, 4, graphmodel.EventKindStop),
			},
			exp: expected{
				event: event(3, 22, graphmodel.EventKindStart),
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

func TestDefinitionEventNextOccurenceAfter(t *testing.T) {
	type expected struct {
		next time.Time
	}
	tests := map[string]struct {
		event DefinitionEvent
		after time.Time
		exp   expected
	}{
		"+day,+hour": {
			event: event(4, 22, graphmodel.EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 17, 22, 0, 0, 0, time.UTC),
			},
		},
		"++day,+hour": {
			event: event(6, 22, graphmodel.EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 19, 22, 0, 0, 0, time.UTC),
			},
		},
		"++day,-hour": {
			event: event(6, 16, graphmodel.EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 19, 16, 0, 0, 0, time.UTC),
			},
		},
		"day,+hour": {
			event: event(3, 22, graphmodel.EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 16, 22, 0, 0, 0, time.UTC),
			},
		},
		"-day,+hour": {
			event: event(2, 22, graphmodel.EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 22, 22, 0, 0, 0, time.UTC),
			},
		},
		"--day,+hour": {
			event: event(0, 22, graphmodel.EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 20, 22, 0, 0, 0, time.UTC),
			},
		},
		"--day,-hour": {
			event: event(0, 16, graphmodel.EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 20, 16, 0, 0, 0, time.UTC),
			},
		},
		"day,hour": {
			event: event(3, 19, graphmodel.EventKindStart),
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 23, 19, 0, 0, 0, time.UTC),
			},
		},
		"+day,-hour,+minute": {
			event: event(4, 16, graphmodel.EventKindStart),
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

func event(day time.Weekday, hour uint8, kind graphmodel.EventKind) DefinitionEvent {
	return DefinitionEvent{
		Weekday:   day,
		Hour:      hour,
		EventKind: kind,
	}
}
