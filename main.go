package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

func handleTick(s Server) {
	http.HandleFunc("/tick", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("handling /tick")
		// Unmarshal and validate request
		if r.Method != "POST" {
			http.Error(w, "must use POST to access /tick", http.StatusMethodNotAllowed)
			return
		}

		var req TickRequest
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
	})
}

func handleGetIntervals(s Server) {
	http.HandleFunc("/intervals", func(w http.ResponseWriter, r *http.Request) {
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
		req := GetIntervalsRequest{
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
	})
}

func handleClear(s Server) {
	http.HandleFunc("/clear", func(w http.ResponseWriter, r *http.Request) {
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
	})
}

func handleToday(s Server) {
	http.HandleFunc("/today", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("handling /today")
		// Unmarshal and validate request
		if r.Method != "GET" {
			http.Error(w, "must use GET to access /today", http.StatusMethodNotAllowed)
			return
		}
		s.GetToday(w)
		return
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

func startServing(c Clock, file string) {
	s, err := NewServer(c, file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not start server: %v", err)
		os.Exit(1)
	}
	handleTick(s)
	handleGetIntervals(s)
	handleToday(s)
	handleClear(s)

	// Return to non-endpoint calls with 404
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	// Start serving requests
	log.Fatal(http.ListenAndServe(":10101", nil))
}

func main() {
	startServing(SystemClock{}, "" /* default db file */)
}
