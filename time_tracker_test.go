package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

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

func fatal(t *testing.T, tmpl string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	finalArgs := []interface{}{path.Base(file), line}
	finalArgs = append(finalArgs, args...)
	t.Fatalf("\n%s:%d: "+tmpl, finalArgs...)
}

func CheckNil(t *testing.T, err error) {
	if err != nil {
		fatal(t, "expected <nil> error, but was:\n	%v", err)
	}
}

func CheckEq(t *testing.T, actual interface{}, expected interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		fatal(t, "expected: \"%+v\"\nbut was: \"%+v\"", expected, actual)
	}
}

func ReadBody(t *testing.T, resp *http.Response) string {
	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(resp.Body)
	CheckNil(t, err)
	return buf.String()
}

// TestParsing does a basic test of the TimeTracker API (registering 4 ticks
// that create two intervals
func TestParsing(t *testing.T) {
	start := time.Date(
		/* date */ 2017, 7, 1,
		/* time */ 12, 0, 0,
		/* nsec, location */ 0, time.UTC)
	testClock.Set(start)

	// Make several calls to /tick via the HTTP API (simulating that they arrive
	// 'intervalMinutes' apart, so that there are two distinct intervals here)
	client := &http.Client{}
	for _, intervalMinutes := range []int64{0, 1, 1, 30, 1} {
		interval := time.Duration(intervalMinutes * int64(time.Minute))
		testClock.Add(interval)
		req, err := http.NewRequest("POST", "http://localhost:10101/tick",
			strings.NewReader(`{ "labels": ["label1", "label2"]}`))
		// strings.NewReader(`
		// 	{
		// 		"labels": [ "label1"
		// 		          , "label2"
		// 							]
		// 	}
		// `))
		CheckNil(t, err)
		resp, err := client.Do(req)
		CheckNil(t, err)
		CheckEq(t, resp.StatusCode, http.StatusOK)
		b := ReadBody(t, resp)
		CheckEq(t, b, "")
	}

	// Make a call to /get-intervals and make sure the two expected intervals
	// are returned
	for _, label := range []string{"label1", "label2"} {
		req, err := http.NewRequest("GET", "http://localhost:10101/intervals",
			strings.NewReader(`
				{
					"label": "`+label+`"
				}
			`))
		CheckNil(t, err)
		resp, err := client.Do(req)
		CheckNil(t, err)
		CheckEq(t, resp.StatusCode, http.StatusOK)

		decoder := json.NewDecoder(resp.Body)
		var actual GetIntervalsResponse
		decoder.Decode(&actual)
		CheckEq(t, actual, GetIntervalsResponse{
			Intervals: []Interval{
				{Start: start,
					End: start.Add(2 * time.Minute)},
				{Start: start.Add(32 * time.Minute),
					End: start.Add(33 * time.Minute)},
			},
		})
	}
}

func TestMain(m *testing.M) {
	go startServing(testClock)
	os.Exit(m.Run())
}
