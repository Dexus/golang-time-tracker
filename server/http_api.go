package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/msteffen/golang-time-tracker/pkg/api"
)

var c api.Clock

type httpAPIServer struct {
	api.APIServer
}

func (s httpAPIServer) tick(w http.ResponseWriter, r *http.Request) {
	log.Printf("handling /tick")
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
	log.Printf("handling /intervals")
	// Unmarshal and validate request
	if r.Method != "GET" {
		http.Error(w, "must use GET to access /intervals", http.StatusMethodNotAllowed)
		return
	}

	// Trasform GET params into request struct
	var start, end int64 = 0, math.MaxInt64
	var err error
	if startStr := r.URL.Query().Get("start"); startStr != "" {
		start, err = strconv.ParseInt(startStr, 10, 64)
		if err != nil {
			msg := fmt.Sprintf("invalid \"start\" param: %s", err.Error())
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
	}
	if endStr := r.URL.Query().Get("end"); endStr != "" {
		end, err = strconv.ParseInt(r.URL.Query().Get("end"), 10, 64)
		if err != nil {
			msg := fmt.Sprintf("invalid \"end\" param: %s", err.Error())
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
	}
	req := api.GetIntervalsRequest{
		Label: r.URL.Query().Get("label"),
		Start: start,
		End:   end,
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
	log.Printf("handling /clear")
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
	log.Printf("handling /today")
	// Unmarshal and validate request
	if r.Method != "GET" {
		http.Error(w, "must use GET to access /today", http.StatusMethodNotAllowed)
		return
	}
	t := TodayOp{
		server:  s.APIServer,
		clock:   c,
		writer:  w,
		bgWidth: float64(500),
	}
	t.start()
	return
}

// ServeOverHTTP serves the Server API over HTTP, managing HTTP
// reqests/responses
func ServeOverHTTP(server api.APIServer, clock api.Clock) {
	c = clock
	h := httpAPIServer{
		APIServer: server,
	}
	http.HandleFunc("/tick", h.tick)
	http.HandleFunc("/intervals", h.getIntervals)
	http.HandleFunc("/today", h.today)
	http.HandleFunc("/clear", h.clear)

	// Return to non-endpoint calls with 404
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	// Start serving requests
	log.Fatal(http.ListenAndServe(":10101", nil))
}
