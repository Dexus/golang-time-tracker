package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func handleTick(s Server) {
	http.HandleFunc("/tick", func(w http.ResponseWriter, r *http.Request) {
		// Unmarshal and validate request
		if r.Method != "POST" {
			http.Error(w, "must use POST to access /tick", http.StatusMethodNotAllowed)
			return
		}

		d := json.NewDecoder(r.Body)
		var req TickRequest
		err := d.Decode(&req)
		if err != nil {
			http.Error(w, "request did not match expected type", http.StatusBadRequest)
			return
		}

		// Process request
		err = s.Tick(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func handleGetIntervals(s Server) {
	http.HandleFunc("/intervals", func(w http.ResponseWriter, r *http.Request) {
		// Unmarshal and validate request
		if r.Method != "GET" {
			http.Error(w, "must use GET to access /intervals", http.StatusMethodNotAllowed)
			return
		}

		d := json.NewDecoder(r.Body)
		var req GetIntervalsRequest
		err := d.Decode(&req)
		if err != nil {
			http.Error(w, "request did not match expected type", http.StatusBadRequest)
			return
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
		}
		w.WriteHeader(http.StatusOK)
		w.Write(resultJSON)
	})
}

// Clock is an interface wrapping time.Now(), so that clocks can be injected
// into the TimeTracker server for testing
type Clock interface {
	Now() time.Time
}

// SystemClock is the default implementation of the Clock API (in which Now()
// returns time.Now())
type SystemClock struct{}

// Now is SystemClock's implementation of the Clock API (returns time.Now())
func (s SystemClock) Now() time.Time {
	return time.Now()
}

func startServing(c Clock) {
	s := NewServer(c)
	handleTick(s)
	handleGetIntervals(s)

	// Return to non-endpoint calls with 404
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	// Start serving requests
	log.Fatal(http.ListenAndServe(":10101", nil))
}

func main() {
	startServing(SystemClock{})
}
