package api

import "time"

// Clock is an interface wrapping time.Now(), so that clocks can be injected
// into the TimeTracker server for testing
type Clock interface {
	Now() time.Time
}

// SystemClock is the default implementation of the Clock API (in which Now()
// returns time.Now())
type SystemClock struct{}

// Now is SystemClock's implementation of the Clock API (returns time.Now())
func (s SystemClock) Now() time.Time {
	return time.Now()
}

// TestingClock is an implementation of the Clock API that's useful for testing
type TestingClock struct {
	time.Time
}

// Now returns the current time according to 't'
func (t *TestingClock) Now() time.Time {
	return t.Time
}

var testClock = &TestingClock{
	Time: time.Time{},
}

// Add advances 't' by the duration 'd'
func (t *TestingClock) Add(d time.Duration) {
	t.Time = t.Time.Add(d)
}

// Set sets the current time in 't' to 'to'
func (t *TestingClock) Set(to time.Time) {
	t.Time = to
}
