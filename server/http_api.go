package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/msteffen/golang-time-tracker/api"
	cu "github.com/msteffen/golang-time-tracker/clientutil"
	"github.com/msteffen/golang-time-tracker/webui"
)

type httpAPIServer struct {
	// Not Owned
	clock api.Clock // not owned b/c time isn't modified, but is read by /today

	// Owned
	api.APIServer // Unclear if this is owned or not
	startTime     time.Time
}

func (s httpAPIServer) tick(w http.ResponseWriter, r *http.Request) {
	glog.Infof("handling /tick")
	// Unmarshal and validate request
	if r.Method != "POST" {
		http.Error(w, "must use POST to access /tick", http.StatusMethodNotAllowed)
		return
	}

	var req api.TickRequest
	d := json.NewDecoder(r.Body)
	if err := d.Decode(&req); err != nil {
		msg := fmt.Sprintf("request did not match expected type: %v", err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// Process request
	err := s.Tick(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s httpAPIServer) getIntervals(w http.ResponseWriter, r *http.Request) {
	glog.Infof("handling /intervals")
	// Unmarshal and validate request
	if r.Method != "GET" {
		http.Error(w, "must use GET to access /intervals", http.StatusMethodNotAllowed)
		return
	}

	// Trasform GET params into request struct
	boundary := []int64{0, math.MaxInt64} // start and end
	var err error
	for i, param := range []string{"start", "end"} {
		if s := r.URL.Query().Get(param); s != "" {
			boundary[i], err = strconv.ParseInt(s, 10, 64)
			if err != nil {
				msg := fmt.Sprintf("invalid \"%s\" value: %s", param, err.Error())
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
		}
	}
	req := api.GetIntervalsRequest{
		Start: boundary[0],
		End:   boundary[1],
	}

	// Process request
	result, err := s.GetIntervals(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "could not serialize result: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(resultJSON)
}

func (s httpAPIServer) clear(w http.ResponseWriter, r *http.Request) {
	glog.Infof("handling /clear")
	// Unmarshal and validate request
	if r.Method != "POST" {
		http.Error(w, "must use POST to access /clear", http.StatusMethodNotAllowed)
		return
	}

	// Require a body to ensure that I can't accidentally clear from my browser
	req := make(map[string]interface{})
	d := json.NewDecoder(r.Body)
	if err := d.Decode(&req); err != nil {
		msg := fmt.Sprintf("request did not match expected type: %v", err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	if req["confirm"] != "yes" {
		http.Error(w, "Must send confirmation message to delete all server data", http.StatusBadRequest)
		return
	}
	if err := s.Clear(); err != nil {
		http.Error(w, fmt.Sprintf("Could not clear DB: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetToday writes the http response for the /today page to 'w'.
func (s httpAPIServer) today(w http.ResponseWriter, r *http.Request) {
	glog.Infof("handling /today")
	// Unmarshal and validate request
	if r.Method != "GET" {
		http.Error(w, "must use GET to access /today", http.StatusMethodNotAllowed)
		return
	}
	t := webui.TodayOp{
		Server:  s.APIServer,
		Clock:   s.clock,
		Writer:  w,
		BgWidth: float64(500),
	}
	t.Start()
	return
}

func (s httpAPIServer) status(w http.ResponseWriter, r *http.Request) {
	glog.Infof("handling /status")
	// Unmarshal and validate request
	if r.Method != "GET" {
		http.Error(w, "must use GET to access /status", http.StatusMethodNotAllowed)
		return
	}
	w.Write([]byte(time.Now().Sub(s.startTime).String()))
}

type loggingHandler struct {
	mux *http.ServeMux
}

func (h loggingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	glog.Infof("HTTP request: %#v", req)
	h.mux.ServeHTTP(w, req)
}

// serveOverHTTP serves the Server API over HTTP, managing HTTP
// reqests/responses
func ServeOverHTTP(socketPath string, clock api.Clock, server api.APIServer) error {
	// Stat socket file
	info, err := os.Stat(socketPath)
	glog.Infof("socket stat (socket should not exist):\ninfo = %#v\nerr = %v\n", info, err)

	// Socket exists: return error indicating that server can't start
	if err == nil {
		// Socket exists, but is unexpected file type. Don't remove it in case it
		// belongs to another application somehow
		if info.Mode()&os.ModeType != os.ModeSocket {
			glog.Fatalf("socket file had unexpected file type: %s (maybe it's owned by another application?)", info.Mode())
		}

		// Socket exists; see if server is running by sending request
		_, err = cu.GetClient(socketPath).Get("/status")
		if err == nil {
			glog.Fatal("time-tracker is already running")
		}
		glog.Fatalf("time-tracker is already running, but not responding. Try: 'lsof %s'", socketPath)
	}

	h := httpAPIServer{
		clock:     clock,
		APIServer: server,
		startTime: time.Now(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc(socketPath+"/status", h.status)
	mux.HandleFunc(socketPath+"/tick", h.tick)
	mux.HandleFunc(socketPath+"/intervals", h.getIntervals)
	mux.HandleFunc(socketPath+"/today", h.today)
	mux.HandleFunc(socketPath+"/clear", h.clear)
	mux.Handle(socketPath, http.NotFoundHandler()) // Return to non-endpoint calls with 404

	// Start serving requests
	s := http.Server{
		Handler: loggingHandler{mux: mux},
	}

	glog.Infof("http server about to listen on %s", socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("could not listen on unix socket at %s: %v", socketPath, err)
	}
	return s.Serve(listener)
}
