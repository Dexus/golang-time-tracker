package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"

	"github.com/msteffen/golang-time-tracker/api"
	tu "github.com/msteffen/golang-time-tracker/testutil"
)

// testDir is initialized in TestMain--it contains db files and sockets for
// every test
var testDir string

// TestParsing does a basic test of the TimeTracker API (registering 4 ticks
// that create two intervals
func TestParsing(t *testing.T) {
	s := StartTestServer(t, testDir)
	t.Log("Test server started")
	ts := time.Date(
		/* date */ 2017, 7, 1,
		/* time */ 12, 0, 0,
		/* nsec, location */ 0, time.Local)
	s.Set(ts)

	// Make several calls to /tick via the HTTP API (simulating that they arrive
	// several minutes apart, so that there are two distinct intervals here).
	// Don't use TickAt, to test json parsing.
	for _, i := range []int64{0, 1, 1, 30, 1} {
		s.Add(time.Duration(i * int64(time.Minute)))
		resp, err := s.PostString("/tick", `{"label":"work"}`)
		tu.Check(t,
			tu.Nil(err),
			tu.Eq(ReadBody(t, resp), ""),
			tu.Eq(resp.StatusCode, http.StatusOK),
		)
	}

	// Make a call to /intervals and make sure the two expected intervals
	// are returned
	morning := time.Date(2017, 7, 1, 0, 0, 0, 0, time.Local)
	night := morning.Add(24 * time.Hour)
	url := fmt.Sprintf("/intervals?start=%d&end=%d", morning.Unix(), night.Unix())
	resp, err := s.Get(url)
	tu.Check(t,
		tu.Nil(err),
		tu.Eq(resp.StatusCode, http.StatusOK),
	)

	buf := &bytes.Buffer{}
	buf.ReadFrom(resp.Body)
	t.Logf("Response body:\n%s\n", buf.String())

	var actual api.GetIntervalsResponse
	decoder := json.NewDecoder(buf)
	decoder.Decode(&actual)
	tu.Check(t, tu.Eq(actual, api.GetIntervalsResponse{
		Intervals: []api.Interval{
			{
				Start: ts.Unix(),
				End:   ts.Add(2 * time.Minute).Unix(),
			},
			{
				Start: ts.Add(32 * time.Minute).Unix(),
				End:   ts.Add(33 * time.Minute).Unix(),
			},
		},
	}))
}

// TestGetIntervalsBoundary checks that GetIntervals only returns intervals
// within the given time range
func TestGetIntervalsBoundary(t *testing.T) {
	s := StartTestServer(t, testDir)
	ts := time.Date(
		/* date */ 2017, 7, 1,
		/* time */ 6, 0, 0,
		/* nsec, location */ 0, time.Local)
	s.Set(ts)

	// tick every 20 minutes for 12 hours, so we have a single interval from 6am
	// to 6pm
	hours, ticksPerHour := 12, 3
	s.TickAt("work", 0)
	for i := 0; i < (hours * ticksPerHour); i++ {
		s.TickAt("work", 20)
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
	expected := [][]api.Interval{
		// no overlap
		{},
		// end at noon (req end)
		{{Start: ts.Unix(), End: ts.Add(6 * time.Hour).Unix()}},
		// full interval
		{{Start: ts.Unix(), End: ts.Add(12 * time.Hour).Unix()}},
		// begin at noon (req start)
		{{Start: ts.Add(6 * time.Hour).Unix(), End: ts.Add(12 * time.Hour).Unix()}},
		// no overlap
		{},
	}

	// Make a call to /get-intervals and make sure the two expected intervals
	// are returned
	for i := 0; i < len(name); i++ {
		t.Run(name[i], func(t *testing.T) {
			reqStart, reqEnd := reqStartTs[i], reqStartTs[i].Add(24*time.Hour)
			url := fmt.Sprintf("/intervals?start=%d&end=%d", reqStart.Unix(), reqEnd.Unix())
			resp, err := s.Get(url)
			tu.Check(t,
				tu.Nil(err),
				tu.Eq(resp.StatusCode, http.StatusOK),
			)

			var actual api.GetIntervalsResponse
			decoder := json.NewDecoder(resp.Body)
			decoder.Decode(&actual)
			tu.Check(t, tu.Eq(actual, api.GetIntervalsResponse{Intervals: expected[i]}))
		})
	}
}

func TestToday(t *testing.T) {
	s := StartTestServer(t, testDir)
	ts := time.Date(
		/* date */ 2017, 7, 1,
		/* time */ 9, 0, 0,
		/* nsec, location */ 0, time.UTC)
	s.Set(ts)
	s.TickAt("work", 0, 20, 60, 20)

	resp, err := s.Get("/today")
	buf := &bytes.Buffer{}
	buf.ReadFrom(resp.Body)
	t.Log(buf)
	doc, err := html.Parse(buf)
	tu.Check(t, tu.Nil(err))

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
	tu.Check(t, tu.Eq(nIntervals, 2))
}

func TestMain(m *testing.M) {
	// create temporary directory for housing test data
	var err error
	testDir, err = ioutil.TempDir(os.TempDir(), "time-tracker-test-")
	if err != nil {
		panic(fmt.Sprintf("could not create temporary directory for test data: %v", err))
	}
	errCode := m.Run()
	if err := os.RemoveAll(testDir); err != nil {
		panic(fmt.Sprintf("could not remove temp testing directory: %v", err))
	}
	os.Exit(errCode)
}
