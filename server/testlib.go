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

	tu "github.com/msteffen/golang-time-tracker/pkg/testutil"
)

// TestDBFile is the DB file that time-tracker uses when run in tests
const TestDBFile = "test-db"

// SetUpTestServer brings up an in-process time-tracker server serving on
// localhost, for the tests to talk to. If called in TestMain, callers should
// also call defer 'os.Remove(TestDBFile)' right afterwards
func SetUpTestServer() {
	if _, err := os.Stat(TestDBFile); !os.IsNotExist(err) {
		os.Remove(TestDBFile)
	}
	go StartServing(testClock, TestDBFile)
	// Wait until the server is up before proceeding
	req, err := http.NewRequest("GET", "http://localhost:10101/", nil)
	if err != nil {
		panic(fmt.Sprintf("could not create HTTP request: %v", err))
	}
	for {
		_, err := http.DefaultClient.Do(req)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
}

// ReadBody is a helper function that reads resp.Body into a buffer and returns
// it as a string
func ReadBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(resp.Body)
	tu.Check(t, tu.Nil(err))
	return buf.String()
}

// TickAt is a helper function that sends ticks to the local TimeTracker server
// at the given intervals with the given labels
//
// (TickAt(["l1"], 1, 1, 1) would send a tick with the label "l1" at 1 minute
// past start, 2 minutes past start, and 3 minutes past start, logically)
func TickAt(t *testing.T, label string, intervals ...int64) {
	t.Helper()
	request := TickRequest{Label: label}
	var buf bytes.Buffer
	for _, i := range intervals {
		testClock.Add(time.Duration(i * int64(time.Minute)))
		buf.Reset()
		json.NewEncoder(&buf).Encode(request)
		req, err := http.NewRequest("POST", "http://localhost:10101/tick", &buf)
		tu.Check(t, tu.Nil(err))
		resp, err := http.DefaultClient.Do(req)
		tu.Check(t,
			tu.Nil(err),
			tu.Eq(resp.StatusCode, http.StatusOK),
			tu.Eq(ReadBody(t, resp), ""),
		)
	}
}

// ClearData clears all interval data stored in the TimeTracker server, to
// create a fresh environment for each test
func ClearData(t *testing.T) {
	t.Helper()
	req, err := http.NewRequest("POST", "http://localhost:10101/clear",
		strings.NewReader(`{"confirm":"yes"}`))
	tu.Check(t, tu.Nil(err))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	tu.Check(t,
		tu.Nil(err),
		tu.Eq(resp.StatusCode, http.StatusOK),
		tu.Eq(ReadBody(t, resp), ""),
	)
}
