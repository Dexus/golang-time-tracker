// interactive_test.go is not a test in the conventional sense. Its job is to
// load test data into time-tracker and then wait for me to preview the /today
// page. Each "test" loads different data into time-tracker, and is designed to
// be run with 'go test -v . -run TestSituationX\$', where it will load data
// into time-tracker and then wait, serving /today to my browser, so I can
// preview that situation.

package main

import (
	"os"
	"testing"
	"time"
)

func TestTwoIntervals(t *testing.T) {
	if os.Getenv("TIMETRACKER_INTERACTIVE_TESTS") == "" {
		t.Skip("Skip interactive tests during regular testing")
	}
	ClearData(t)
	ts := time.Date(
		/* date */ 2017, 7, 1,
		/* time */ 9, 0, 0,
		/* nsec, location */ 0, time.UTC)
	testClock.Set(ts)
	TickAt(t, "", 0, 20, 60, 20)
	testClock.Add(5)
	time.Sleep(12 * time.Hour)
}
