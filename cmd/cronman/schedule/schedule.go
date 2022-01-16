package schedule

import (
	"time"
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
