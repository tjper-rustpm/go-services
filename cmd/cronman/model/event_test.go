package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventsNextEvent(t *testing.T) {
	type expected struct {
		event Event
		when  time.Time
	}
	tests := map[string]struct {
		dt     time.Time
		kind   EventKind
		events Events
		exp    expected
	}{
		"daily start stop": {
			dt:   time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			kind: EventKindStart,
			events: Events{
				{Schedule: "0 22 * * *", Kind: EventKindStart},
				{Schedule: "0 4 * * *", Kind: EventKindStop},
			},
			exp: expected{
				event: Event{Schedule: "0 22 * * *", Kind: EventKindStart},
				when:  time.Date(2020, time.September, 16, 22, 0, 0, 0, time.UTC),
			},
		},
		"daily start live stop": {
			dt:   time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			kind: EventKindLive,
			events: Events{
				{Schedule: "30 20 * * *", Kind: EventKindStart},
				{Schedule: "0 21 * * *", Kind: EventKindLive},
				{Schedule: "0 6 * * *", Kind: EventKindStop},
			},
			exp: expected{
				event: Event{Schedule: "0 21 * * *", Kind: EventKindLive},
				when:  time.Date(2020, time.September, 16, 21, 0, 0, 0, time.UTC),
			},
		},
		"daily skip stop": {
			dt:   time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			kind: EventKindStart,
			events: Events{
				{Schedule: "0 14 * * *", Kind: EventKindStart},
				{Schedule: "0 4 * * *", Kind: EventKindStop},
			},
			exp: expected{
				event: Event{Schedule: "0 14 * * *", Kind: EventKindStart},
				when:  time.Date(2020, time.September, 17, 14, 0, 0, 0, time.UTC),
			},
		},
		"daily start stop, weekly mapwipe": {
			dt:   time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			kind: EventKindMapWipe,
			events: Events{
				{Schedule: "0 20 * * *", Kind: EventKindStart},
				{Schedule: "0 6 * * *", Kind: EventKindStop},
				{Schedule: "0 18 * * *", Weekday: weekday(time.Thursday), Kind: EventKindMapWipe},
			},
			exp: expected{
				event: Event{Schedule: "0 18 * * *", Weekday: weekday(time.Thursday), Kind: EventKindMapWipe},
				when:  time.Date(2020, time.September, 17, 18, 0, 0, 0, time.UTC),
			},
		},
		"daily start stop, bi-weekly mapwipe": {
			dt:   time.Date(2020, time.September, 4, 19, 0, 0, 0, time.UTC),
			kind: EventKindMapWipe,
			events: Events{
				{Schedule: "0 20 * * *", Kind: EventKindStart},
				{Schedule: "0 6 * * *", Kind: EventKindStop},
				{Schedule: "0 18 15-21 * *", Weekday: weekday(time.Thursday), Kind: EventKindMapWipe},
			},
			exp: expected{
				event: Event{Schedule: "0 18 15-21 * *", Weekday: weekday(time.Thursday), Kind: EventKindMapWipe},
				when:  time.Date(2020, time.September, 17, 18, 0, 0, 0, time.UTC),
			},
		},
		"daily start stop, bi-weekly 2nd mapwipe": {
			dt:   time.Date(2020, time.October, 1, 18, 0, 0, 0, time.UTC),
			kind: EventKindMapWipe,
			events: Events{
				{Schedule: "0 20 * * *", Kind: EventKindStart},
				{Schedule: "0 6 * * *", Kind: EventKindStop},
				{Schedule: "0 18 28-31 * *", Weekday: weekday(time.Thursday), Kind: EventKindMapWipe},
			},
			exp: expected{
				event: Event{Schedule: "0 18 28-31 * *", Weekday: weekday(time.Thursday), Kind: EventKindMapWipe},
				when:  time.Date(2020, time.October, 29, 18, 0, 0, 0, time.UTC),
			},
		},
		"daily start stop, weekly mapwipe, bi-weekly fullwipe, october": {
			dt:   time.Date(2020, time.October, 1, 18, 0, 0, 0, time.UTC),
			kind: EventKindFullWipe,
			events: Events{
				{Schedule: "0 20 * * *", Kind: EventKindStart},
				{Schedule: "0 6 * * *", Kind: EventKindStop},
				{Schedule: "0 18 8-14 * *", Weekday: weekday(time.Thursday), Kind: EventKindMapWipe},
				{Schedule: "0 18 22-28 * *", Weekday: weekday(time.Thursday), Kind: EventKindMapWipe},
				{Schedule: "0 18 1-7 * *", Weekday: weekday(time.Thursday), Kind: EventKindFullWipe},
				{Schedule: "0 18 15-21 * *", Weekday: weekday(time.Thursday), Kind: EventKindFullWipe},
			},
			exp: expected{
				event: Event{Schedule: "0 18 15-21 * *", Weekday: weekday(time.Thursday), Kind: EventKindFullWipe},
				when:  time.Date(2020, time.October, 15, 18, 0, 0, 0, time.UTC),
			},
		},
		"daily start stop, weekly mapwipe, bi-weekly fullwipe, november": {
			dt:   time.Date(2020, time.October, 15, 18, 0, 0, 0, time.UTC),
			kind: EventKindFullWipe,
			events: Events{
				{Schedule: "0 20 * * *", Kind: EventKindStart},
				{Schedule: "0 6 * * *", Kind: EventKindStop},
				{Schedule: "0 18 8-14 * *", Weekday: weekday(time.Thursday), Kind: EventKindMapWipe},
				{Schedule: "0 18 22-28 * *", Weekday: weekday(time.Thursday), Kind: EventKindMapWipe},
				{Schedule: "0 18 1-7 * *", Weekday: weekday(time.Thursday), Kind: EventKindFullWipe},
				{Schedule: "0 18 15-21 * *", Weekday: weekday(time.Thursday), Kind: EventKindFullWipe},
			},
			exp: expected{
				event: Event{Schedule: "0 18 1-7 * *", Weekday: weekday(time.Thursday), Kind: EventKindFullWipe},
				when:  time.Date(2020, time.November, 5, 18, 0, 0, 0, time.UTC),
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			event, when, err := test.events.NextEvent(test.dt, test.kind)
			assert.Nil(t, err)
			assert.Equal(t, test.exp.event, *event)
			assert.Equal(t, test.exp.when, *when)
		})
	}
}

func TestEventNext(t *testing.T) {
	type expected struct {
		next time.Time
	}
	tests := map[string]struct {
		event Event
		after time.Time
		exp   expected
	}{
		"daily": {
			event: Event{Schedule: "0 22 * * *", Kind: EventKindStart},
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 16, 22, 0, 0, 0, time.UTC),
			},
		},
		"weekly": {
			event: Event{Schedule: "0 4 * * *", Weekday: weekday(time.Friday), Kind: EventKindStart},
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 18, 4, 0, 0, 0, time.UTC),
			},
		},
		"1st week": {
			event: Event{Schedule: "0 4 1-7 * *", Weekday: weekday(time.Friday), Kind: EventKindStart},
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.October, 2, 4, 0, 0, 0, time.UTC),
			},
		},
		"2nd week": {
			event: Event{Schedule: "0 4 8-14 * *", Weekday: weekday(time.Friday), Kind: EventKindStart},
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.October, 9, 4, 0, 0, 0, time.UTC),
			},
		},
		"3rd week": {
			event: Event{Schedule: "0 4 15-21 * *", Weekday: weekday(time.Friday), Kind: EventKindStart},
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 18, 4, 0, 0, 0, time.UTC),
			},
		},
		"4th week": {
			event: Event{Schedule: "0 4 22-28 * *", Weekday: weekday(time.Friday), Kind: EventKindStart},
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			exp: expected{
				next: time.Date(2020, time.September, 25, 4, 0, 0, 0, time.UTC),
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			next, err := test.event.Next(test.after)
			assert.Nil(t, err)
			assert.True(
				t,
				test.exp.next.Equal(next),
				"exp: %s\nnext: %s",
				test.exp.next.Format(time.RFC822),
				next.Format(time.RFC822),
			)
		})
	}
}

func TestEventOccurrences(t *testing.T) {
	type expected struct {
		occurrences []time.Time
	}
	tests := map[string]struct {
		event Event
		after time.Time
		until time.Time
		exp   expected
	}{
		"daily": {
			event: Event{Schedule: "30 21 * * *", Kind: EventKindStart},
			after: time.Date(2020, time.September, 16, 19, 0, 0, 0, time.UTC),
			until: time.Date(2020, time.September, 23, 19, 0, 0, 0, time.UTC),
			exp: expected{
				occurrences: []time.Time{
					time.Date(2020, time.September, 16, 21, 30, 0, 0, time.UTC),
					time.Date(2020, time.September, 17, 21, 30, 0, 0, time.UTC),
					time.Date(2020, time.September, 18, 21, 30, 0, 0, time.UTC),
					time.Date(2020, time.September, 19, 21, 30, 0, 0, time.UTC),
					time.Date(2020, time.September, 20, 21, 30, 0, 0, time.UTC),
					time.Date(2020, time.September, 21, 21, 30, 0, 0, time.UTC),
					time.Date(2020, time.September, 22, 21, 30, 0, 0, time.UTC),
				},
			},
		},
		"weekly": {
			event: Event{Schedule: "30 21 * * *", Weekday: weekday(time.Thursday), Kind: EventKindStart},
			after: time.Date(2020, time.September, 1, 0, 0, 0, 0, time.UTC),
			until: time.Date(2020, time.September, 31, 23, 0, 0, 0, time.UTC),
			exp: expected{
				occurrences: []time.Time{
					time.Date(2020, time.September, 3, 21, 30, 0, 0, time.UTC),
					time.Date(2020, time.September, 10, 21, 30, 0, 0, time.UTC),
					time.Date(2020, time.September, 17, 21, 30, 0, 0, time.UTC),
					time.Date(2020, time.September, 24, 21, 30, 0, 0, time.UTC),
					time.Date(2020, time.September, 31, 21, 30, 0, 0, time.UTC),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			occurrences, err := test.event.Occurrences(test.after, test.until)
			assert.Nil(t, err)
			assert.Equal(t, test.exp.occurrences, occurrences)
		})
	}
}

// --- helpers ---

func weekday(v time.Weekday) *time.Weekday { return &v }
