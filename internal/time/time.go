// Package time provided and API for interaction with time. This package
// primarily wraps the standard library time package in structures to make
// the usage of time-related data-types mockable.
package time

import "time"

// Time wraps time-related functionality from the standard library to enable
// mocking in tests.
type Time struct{}

// Now wraps time.Now.
func (t Time) Now() time.Time {
	return time.Now()
}

// Until wraps time.Until.
func (t Time) Until(when time.Time) time.Duration {
	return time.Until(when)
}

// NewMock initializes a new Mock instance.
func NewMock(now time.Time) *Mock {
	return &Mock{now: now}
}

// Mock may be used to mock the functionality provided by Time.
type Mock struct {
	now   time.Time
	until time.Duration
}

// Now retrieves the mocked time.Now value.
func (m Mock) Now() time.Time {
	return m.now
}

// SetUntil sets the value to be returned by Mock.Until.
func (m *Mock) SetUntil(until time.Duration) {
	m.until = until
}

// Until mocks Time.Until by returning the value set by SetUntil.
func (m Mock) Until(_ time.Time) time.Duration {
	return m.until
}
