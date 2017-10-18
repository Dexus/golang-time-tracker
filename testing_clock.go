package main

import "time"

// TestingClock is an implementation of the Clock API that's useful for testing
type TestingClock struct {
	*time.Time
}

func (t TestingClock) Now() time.Time {
	return *t.Time
}

var testClock = TestingClock{
	Time: new(time.Time),
}

func (t TestingClock) Add(d time.Duration) {
	*t.Time = t.Time.Add(d)
}

func (t TestingClock) Set(to time.Time) {
	*t.Time = to
}

func (t TestingClock) Advance(d time.Duration) {
	t.Time.Add(d)
}
