package schedule

import (
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/graph/model"

	"github.com/stretchr/testify/assert"
)

func TestIsNthWeekDay(t *testing.T) {
	type expected struct {
		is bool
	}
	tests := map[string]struct {
		t       time.Time
		week    Week
		weekDay time.Weekday
		exp     expected
	}{
		"april-1":    {t: date(time.April, 1), week: FirstWeek, weekDay: time.Thursday, exp: expected{is: true}},
		"april-2":    {t: date(time.April, 2), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-3":    {t: date(time.April, 3), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-4":    {t: date(time.April, 4), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-5":    {t: date(time.April, 5), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-6":    {t: date(time.April, 6), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-7":    {t: date(time.April, 7), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-8":    {t: date(time.April, 8), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: true}},
		"april-9":    {t: date(time.April, 9), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-10":   {t: date(time.April, 10), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-11":   {t: date(time.April, 11), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-12":   {t: date(time.April, 12), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-13":   {t: date(time.April, 13), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-14":   {t: date(time.April, 14), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-15":   {t: date(time.April, 15), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: true}},
		"april-16":   {t: date(time.April, 16), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-17":   {t: date(time.April, 17), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-18":   {t: date(time.April, 18), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-19":   {t: date(time.April, 19), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-20":   {t: date(time.April, 20), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-21":   {t: date(time.April, 21), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-22":   {t: date(time.April, 22), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: true}},
		"april-23":   {t: date(time.April, 23), week: FifthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-24":   {t: date(time.April, 24), week: FifthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-25":   {t: date(time.April, 25), week: FifthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-26":   {t: date(time.April, 26), week: FifthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-27":   {t: date(time.April, 27), week: FifthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-28":   {t: date(time.April, 28), week: FifthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"april-29":   {t: date(time.April, 29), week: FifthWeek, weekDay: time.Thursday, exp: expected{is: true}},
		"april-30":   {t: date(time.April, 30), week: FirstWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-1":  {t: date(time.October, 1), week: FirstWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-2":  {t: date(time.October, 2), week: FirstWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-3":  {t: date(time.October, 3), week: FirstWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-4":  {t: date(time.October, 4), week: FirstWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-5":  {t: date(time.October, 5), week: FirstWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-6":  {t: date(time.October, 6), week: FirstWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-7":  {t: date(time.October, 7), week: FirstWeek, weekDay: time.Thursday, exp: expected{is: true}},
		"october-8":  {t: date(time.October, 8), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-9":  {t: date(time.October, 9), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-10": {t: date(time.October, 10), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-11": {t: date(time.October, 11), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-12": {t: date(time.October, 12), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-13": {t: date(time.October, 13), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-14": {t: date(time.October, 14), week: SecondWeek, weekDay: time.Thursday, exp: expected{is: true}},
		"october-15": {t: date(time.October, 15), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-16": {t: date(time.October, 16), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-17": {t: date(time.October, 17), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-18": {t: date(time.October, 18), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-19": {t: date(time.October, 19), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-20": {t: date(time.October, 20), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-21": {t: date(time.October, 21), week: ThirdWeek, weekDay: time.Thursday, exp: expected{is: true}},
		"october-22": {t: date(time.October, 22), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-23": {t: date(time.October, 23), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-24": {t: date(time.October, 24), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-25": {t: date(time.October, 25), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-26": {t: date(time.October, 26), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-27": {t: date(time.October, 27), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-28": {t: date(time.October, 28), week: FourthWeek, weekDay: time.Thursday, exp: expected{is: true}},
		"october-29": {t: date(time.October, 29), week: FifthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-30": {t: date(time.October, 30), week: FifthWeek, weekDay: time.Thursday, exp: expected{is: false}},
		"october-31": {t: date(time.October, 31), week: FifthWeek, weekDay: time.Thursday, exp: expected{is: false}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.exp.is, IsNthWeekDay(test.t, test.week, test.weekDay))
		})
	}
}

func TestIsWipe(t *testing.T) {
	type expected struct {
		isMapWipe  bool
		isFullWipe bool
	}
	tests := map[string]struct {
		t                      time.Time
		mapWipeFrequency       model.WipeFrequency
		blueprintWipeFrequency model.WipeFrequency
		exp                    expected
	}{
		"april-1-weekly-monthly": {
			t:                      date(time.April, 1),
			mapWipeFrequency:       model.WipeFrequencyWeekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: true,
			},
		},
		"april-2-weekly-monthly": {
			t:                      date(time.April, 2),
			mapWipeFrequency:       model.WipeFrequencyWeekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
		"april-8-weekly-monthly": {
			t:                      date(time.April, 8),
			mapWipeFrequency:       model.WipeFrequencyWeekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  true,
				isFullWipe: false,
			},
		},
		"april-9-weekly-monthly": {
			t:                      date(time.April, 9),
			mapWipeFrequency:       model.WipeFrequencyWeekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
		"april-15-weekly-monthly": {
			t:                      date(time.April, 15),
			mapWipeFrequency:       model.WipeFrequencyWeekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  true,
				isFullWipe: false,
			},
		},
		"april-16-weekly-monthly": {
			t:                      date(time.April, 16),
			mapWipeFrequency:       model.WipeFrequencyWeekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
		"april-22-weekly-monthly": {
			t:                      date(time.April, 22),
			mapWipeFrequency:       model.WipeFrequencyWeekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  true,
				isFullWipe: false,
			},
		},
		"april-23-weekly-monthly": {
			t:                      date(time.April, 23),
			mapWipeFrequency:       model.WipeFrequencyWeekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
		"april-29-weekly-monthly": {
			t:                      date(time.April, 29),
			mapWipeFrequency:       model.WipeFrequencyWeekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
		"april-30-weekly-monthly": {
			t:                      date(time.April, 30),
			mapWipeFrequency:       model.WipeFrequencyWeekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
		"april-1-biweekly-monthly": {
			t:                      date(time.April, 1),
			mapWipeFrequency:       model.WipeFrequencyBiweekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: true,
			},
		},
		"april-8-biweekly-monthly": {
			t:                      date(time.April, 8),
			mapWipeFrequency:       model.WipeFrequencyBiweekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
		"april-15-biweekly-monthly": {
			t:                      date(time.April, 15),
			mapWipeFrequency:       model.WipeFrequencyBiweekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  true,
				isFullWipe: false,
			},
		},
		"april-22-biweekly-monthly": {
			t:                      date(time.April, 22),
			mapWipeFrequency:       model.WipeFrequencyBiweekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
		"april-29-biweekly-monthly": {
			t:                      date(time.April, 29),
			mapWipeFrequency:       model.WipeFrequencyBiweekly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
		"april-1-monthly-monthly": {
			t:                      date(time.April, 1),
			mapWipeFrequency:       model.WipeFrequencyMonthly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: true,
			},
		},
		"april-15-monthly-monthly": {
			t:                      date(time.April, 15),
			mapWipeFrequency:       model.WipeFrequencyMonthly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
		"april-29-monthly-monthly": {
			t:                      date(time.April, 29),
			mapWipeFrequency:       model.WipeFrequencyMonthly,
			blueprintWipeFrequency: model.WipeFrequencyMonthly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
		"april-1-biweekly-biweekly": {
			t:                      date(time.April, 1),
			mapWipeFrequency:       model.WipeFrequencyBiweekly,
			blueprintWipeFrequency: model.WipeFrequencyBiweekly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: true,
			},
		},
		"april-8-biweekly-biweekly": {
			t:                      date(time.April, 8),
			mapWipeFrequency:       model.WipeFrequencyBiweekly,
			blueprintWipeFrequency: model.WipeFrequencyBiweekly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
		"april-15-biweekly-biweekly": {
			t:                      date(time.April, 15),
			mapWipeFrequency:       model.WipeFrequencyBiweekly,
			blueprintWipeFrequency: model.WipeFrequencyBiweekly,
			exp: expected{
				isMapWipe:  true,
				isFullWipe: true,
			},
		},
		"april-22-biweekly-biweekly": {
			t:                      date(time.April, 22),
			mapWipeFrequency:       model.WipeFrequencyBiweekly,
			blueprintWipeFrequency: model.WipeFrequencyBiweekly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
		"april-29-biweekly-biweekly": {
			t:                      date(time.April, 29),
			mapWipeFrequency:       model.WipeFrequencyBiweekly,
			blueprintWipeFrequency: model.WipeFrequencyBiweekly,
			exp: expected{
				isMapWipe:  false,
				isFullWipe: false,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			isMapWipe, err := IsMapWipe(test.mapWipeFrequency, time.Thursday, test.t)
			assert.Nil(t, err)
			assert.Equal(t, test.exp.isMapWipe, isMapWipe)

			isFullWipe, err := IsFullWipe(test.blueprintWipeFrequency, time.Thursday, test.t)
			assert.Nil(t, err)
			assert.Equal(t, test.exp.isFullWipe, isFullWipe)
		})
	}
}

// --- helpers ---

func date(month time.Month, day int) time.Time {
	return time.Date(2021, month, day, 0, 0, 0, 0, time.UTC)
}
