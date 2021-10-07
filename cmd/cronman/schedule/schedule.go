package schedule

import (
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/graph/model"
)

// Week in a month.
type Week int

const (
	// FirstWeek of the month
	FirstWeek Week = iota
	// SecondWeek of the month
	SecondWeek
	// ThirdWeek of the month
	ThirdWeek
	// FourthWeek of the month
	FourthWeek
	// FifthWeek of the month
	FifthWeek
)

// IsNthWeekDay determines if the passed time.Time is the weekday in the nth
// week.
func IsNthWeekDay(t time.Time, week Week, day time.Weekday) bool {
	if t.Weekday() != day {
		return false
	}
	var (
		start = int(week) * 7
		end   = start + 7
	)
	if t.Day() < start || t.Day() > end {
		return false
	}
	return true
}

// IsMapWipe determines if the passed time.Time is a "map wipe" day.
func IsMapWipe(
	frequency model.WipeFrequency,
	day time.Weekday,
	t time.Time,
) (bool, error) {
	var wipe bool
	switch frequency {
	case model.WipeFrequencyWeekly:
		wipe = IsNthWeekDay(t, SecondWeek, day) ||
			IsNthWeekDay(t, ThirdWeek, day) ||
			IsNthWeekDay(t, FourthWeek, day)
	case model.WipeFrequencyBiweekly:
		wipe = IsNthWeekDay(t, ThirdWeek, day)
	case model.WipeFrequencyMonthly:
		wipe = false
	default:
		return false, fmt.Errorf("unknown map wipe frequency \"%s\"", frequency)
	}
	return wipe, nil
}

// IsFullWipe determines if the passed time.Time is a "full wipe" time.
func IsFullWipe(
	frequency model.WipeFrequency,
	day time.Weekday,
	t time.Time,
) (bool, error) {
	var wipe bool
	switch frequency {
	case model.WipeFrequencyWeekly:
		wipe = IsNthWeekDay(t, FirstWeek, day) ||
			IsNthWeekDay(t, SecondWeek, day) ||
			IsNthWeekDay(t, ThirdWeek, day) ||
			IsNthWeekDay(t, FourthWeek, day)
	case model.WipeFrequencyBiweekly:
		wipe = IsNthWeekDay(t, FirstWeek, day) ||
			IsNthWeekDay(t, ThirdWeek, day)
	case model.WipeFrequencyMonthly:
		wipe = IsNthWeekDay(t, FirstWeek, day)
	default:
		return false, fmt.Errorf("unknown full wipe frequency \"%s\"", frequency)
	}
	return wipe, nil
}

func WipeDayToWeekDay(day model.WipeDay) (res time.Weekday) {
	switch day {
	case model.WipeDaySunday:
		res = time.Sunday
	case model.WipeDayMonday:
		res = time.Monday
	case model.WipeDayTuesday:
		res = time.Tuesday
	case model.WipeDayWednesday:
		res = time.Wednesday
	case model.WipeDayThursday:
		res = time.Thursday
	case model.WipeDayFriday:
		res = time.Friday
	case model.WipeDaySaturday:
		res = time.Saturday
	}
	return res
}
