package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/msteffen/golang-time-tracker/api"
	cu "github.com/msteffen/golang-time-tracker/clientutil"
	tu "github.com/msteffen/golang-time-tracker/testutil"
)

// ReadBody is a helper function that reads resp.Body into a buffer and returns
// it as a string
func ReadBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(resp.Body)
	tu.Check(t, tu.Nil(err))
	return buf.String()
}

type TestServer struct {
	*testing.T
	*cu.Client
	*api.TestingClock
}

// Bring up an in-process time-tracker server, for the tests to talk to
func StartTestServer(t *testing.T, tmpDir string) TestServer {
	testPC, _, _, ok := runtime.Caller(1)
	if !ok {
		glog.Fatal("could not extract test name")
	}
	testInfo := runtime.FuncForPC(testPC)
	dbPath := path.Join(tmpDir, path.Base(testInfo.Name())+".db")
	glog.Infof("dbPath: %s", dbPath)
	socketPath := path.Join(tmpDir, path.Base(testInfo.Name())+".sock")
	glog.Infof("socketPath: %s\n", socketPath)
	testClock := &api.TestingClock{}

	// Start apiServer and http server
	apiServer, err := api.NewServer(testClock, dbPath)
	if err != nil {
		glog.Fatal("could not create API Server: " + err.Error())
	}
	go ServeOverHTTP(socketPath, testClock, apiServer)

	// Wait until the server is up before proceeding
	client := cu.GetClient(socketPath)
	secs := 60
	for i := 0; i < secs; i++ {
		glog.Infof("waiting until server is up to continue (%d/%d)", i, secs)
		_, err := client.Get("/status")
		if err == nil {
			return TestServer{t, client, testClock}
		}
		time.Sleep(time.Second)
	}
	glog.Fatal(fmt.Sprintf("test server didn't start after %d seconds", secs))
	return TestServer{nil, nil, nil} // never runs
}

// TickAt is a helper function that sends ticks to the local TimeTracker server
// at the given intervals with the given labels
//
// (TickAt(["l1"], 1, 1, 1) would send a tick with the label "l1" at 1 minute
// past start, 2 minutes past start, and 3 minutes past start, logically)
func (s TestServer) TickAt(label string, intervals ...int64) {
	s.T.Helper()
	request := api.TickRequest{Label: label}
	var buf bytes.Buffer
	for _, i := range intervals {
		s.TestingClock.Add(time.Duration(i * int64(time.Minute)))
		buf.Reset()
		json.NewEncoder(&buf).Encode(request)
		resp, err := s.Client.Post("/tick", &buf)
		tu.Check(s.T,
			tu.Nil(err),
			tu.Eq(resp.StatusCode, http.StatusOK),
			tu.Eq(ReadBody(s.T, resp), ""),
		)
	}
}
