package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func handleTick(s Server) {
	http.HandleFunc("/tick", func(w http.ResponseWriter, r *http.Request) {
		// Unmarshal and validate request
		if r.Method != "POST" {
			http.Error(w, "Must use POST to access /tick", http.StatusMethodNotAllowed)
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
	http.HandleFunc("/get-intervals", func(w http.ResponseWriter, r *http.Request) {
		// Unmarshal and validate request
		if r.Method != "GET" {
			http.Error(w, "", http.StatusMethodNotAllowed)
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

func main() {
	s := NewServer()
	handleTick(s)
	handleGetIntervals(s)

	// Return to non-endpoint calls with 404
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("No resource available at %s", r.URL.Path)))
	})

	// Start serving requests
	log.Fatal(http.ListenAndServe(":10101", nil))
}
