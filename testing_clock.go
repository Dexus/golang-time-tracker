package main

import "time"

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
