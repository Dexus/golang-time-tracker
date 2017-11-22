package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

// ReadBody is a helper function that reads resp.Body into a buffer and returns
// it as a string
func ReadBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(resp.Body)
	Check(t, Nil(err))
	return buf.String()
}

// TickAt is a helper function that sends ticks to the local TimeTracker server
// at the given intervals with the given labels
//
// (TickAt(["l1"], 1, 1, 1) would send a tick with the label "l1" at 1 minute
// past start, 2 minutes past start, and 3 minutes past start, logically)
func TickAt(t *testing.T, labels []string, intervals ...int64) {
	t.Helper()
	if labels == nil {
		labels = []string{""}
	}
	request := TickRequest{Labels: labels}
	var buf bytes.Buffer
	for _, i := range intervals {
		testClock.Add(time.Duration(i * int64(time.Minute)))
		buf.Reset()
		json.NewEncoder(&buf).Encode(request)
		req, err := http.NewRequest("POST", "http://localhost:10101/tick", &buf)
		Check(t, Nil(err))
		resp, err := http.DefaultClient.Do(req)
		Check(t,
			Nil(err),
			Eq(resp.StatusCode, http.StatusOK),
			Eq(ReadBody(t, resp), ""),
		)
	}
}

// ClearData clears all interval data stored in the TimeTracker server, to
// create a fresh environment for each test
func ClearData(t *testing.T) {
	t.Helper()
	req, err := http.NewRequest("POST", "http://localhost:10101/clear",
		strings.NewReader(`{"confirm":"yes"}`))
	_, err = http.DefaultClient.Do(req)
	Check(t, Nil(err))
}

// TestParsing does a basic test of the TimeTracker API (registering 4 ticks
// that create two intervals
func TestParsing(t *testing.T) {
	ClearData(t)
	start := time.Date(
		/* date */ 2017, 7, 1,
		/* time */ 12, 0, 0,
		/* nsec, location */ 0, time.Local)
	testClock.Set(start)

	// Make several calls to /tick via the HTTP API (simulating that they arrive
	// several minutes apart, so that there are two distinct intervals here).
	// Don't use TickAt, to test json parsing.
	for _, i := range []int64{0, 1, 1, 30, 1} {
		testClock.Add(time.Duration(i * int64(time.Minute)))
		req, err := http.NewRequest("POST", "http://localhost:10101/tick",
			strings.NewReader(`{"labels":["label1", "label2"]}`))
		Check(t, Nil(err))
		resp, err := http.DefaultClient.Do(req)
		Check(t,
			Nil(err),
			Eq(resp.StatusCode, http.StatusOK),
			Eq(ReadBody(t, resp), ""),
		)
	}

	// Make a call to /get-intervals and make sure the two expected intervals
	// are returned
	for _, label := range []string{"label1", "label2"} {
		morning := time.Date(2017, 7, 1, 0, 0, 0, 0, time.Local)
		night := morning.Add(24 * time.Hour)
		url := fmt.Sprintf("http://localhost:10101/intervals?label=%s&start=%d&end=%d",
			label, morning.Unix(), night.Unix())
		req, err := http.NewRequest("GET", url, nil)
		Check(t, Nil(err))
		resp, err := http.DefaultClient.Do(req)
		Check(t,
			Nil(err),
			Eq(resp.StatusCode, http.StatusOK),
		)

		var actual GetIntervalsResponse
		decoder := json.NewDecoder(resp.Body)
		decoder.Decode(&actual)
		Check(t, Eq(actual, GetIntervalsResponse{
			Intervals: []Interval{
				{Start: start,
					End: start.Add(2 * time.Minute)},
				{Start: start.Add(32 * time.Minute),
					End: start.Add(33 * time.Minute)},
			},
		}))
	}
}

// TestGetIntervalsBoundary checks that GetIntervals only returns intervals
// within the given time range
func TestGetIntervalsBoundary(t *testing.T) {
	ClearData(t)
	start := time.Date(
		/* date */ 2017, 7, 1,
		/* time */ 6, 0, 0,
		/* nsec, location */ 0, time.Local)
	testClock.Set(start)

	// tick every 20 minutes for 12 hours, so we have a single interval from 6am
	// to 6pm
	hours, ticksPerHour := 12, 3
	TickAt(t, nil, 0)
	for i := 0; i < (hours * ticksPerHour); i++ {
		TickAt(t, nil, 20)
	}

	// Enumerate test cases
	name := []string{
		"day-before",
		"overlap-morning",
		"full-day",
		"overlap-evening",
		"day-after",
	}
	reqStartTs := []time.Time{
		time.Date(2017, 6, 30, 0, 0, 0, 0, time.Local),  // day before
		time.Date(2017, 6, 30, 12, 0, 0, 0, time.Local), // overlap morning
		time.Date(2017, 7, 1, 0, 0, 0, 0, time.Local),   // full day
		time.Date(2017, 7, 1, 12, 0, 0, 0, time.Local),  // overlap evening
		time.Date(2017, 7, 2, 0, 0, 0, 0, time.Local),   // day after
	}
	expected := [][]Interval{
		// no overlap
		{},
		// end at noon (req end)
		{{Start: start, End: start.Add(6 * time.Hour)}},
		// full interval
		{{Start: start, End: start.Add(12 * time.Hour)}},
		// begin at noon (req start)
		{{Start: start.Add(6 * time.Hour), End: start.Add(12 * time.Hour)}},
		// no overlap
		{},
	}

	// Make a call to /get-intervals and make sure the two expected intervals
	// are returned
	for i := 0; i < len(name); i++ {
		t.Run(name[i], func(t *testing.T) {
			reqStart, reqEnd := reqStartTs[i], reqStartTs[i].Add(24*time.Hour)
			url := fmt.Sprintf("http://localhost:10101/intervals?start=%d&end=%d",
				reqStart.Unix(), reqEnd.Unix())
			req, err := http.NewRequest("GET", url, nil)
			Check(t, Nil(err))
			resp, err := http.DefaultClient.Do(req)
			Check(t,
				Nil(err),
				Eq(resp.StatusCode, http.StatusOK),
			)

			var actual GetIntervalsResponse
			decoder := json.NewDecoder(resp.Body)
			decoder.Decode(&actual)
			Check(t, Eq(actual, GetIntervalsResponse{Intervals: expected[i]}))
		})
	}
}

func TestToday(t *testing.T) {
	ClearData(t)
	start := time.Date(
		/* date */ 2017, 7, 1,
		/* time */ 9, 0, 0,
		/* nsec, location */ 0, time.UTC)
	testClock.Set(start)
	TickAt(t, nil, 1, 1, 30, 1, 1)

	req, err := http.NewRequest("GET", "http://localhost:10101/today", nil)
	Check(t, Nil(err))
	resp, err := http.DefaultClient.Do(req)
	buf := &bytes.Buffer{}
	buf.ReadFrom(resp.Body)
	doc, err := html.Parse(buf)
	Check(t, Nil(err))

	// Look for the "timefg" elements in document, and make sure there are two of
	// them in the right place
	q := []*html.Node{doc}
	var n *html.Node
	nIntervals := 0
	for len(q) > 0 {
		n, q = q[0], q[1:]
		if n.Type == html.TextNode {
			continue
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			q = append(q, c) // Schedule children
		}
		// Extract "class" attribute if one exists
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, "timefg") {
				nIntervals++
				break
			}
		}
	}
	Check(t, Eq(nIntervals, 2))
}

func TestMain(m *testing.M) {
	go startServing(testClock)
	os.Exit(m.Run())
}
