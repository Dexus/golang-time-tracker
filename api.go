package main

import (
	"time"
)

// Data Structures

// tickDB stores all ticks recorded for each label
type tickDB map[string][]time.Time

// API

// TickRequest is an object sent to the /tick http endpoint, to indicate a file
// save or some other task-related action has occurred
type TickRequest struct {
	// The labels for which we want to issue a tick
	labels []string
}

// GetIntervalsRequest is the object sent to the /get-intervals endpoint.
type GetIntervalsRequest struct {
	// The label whose intervals we want to get
	label string
}

// Interval represents a time interval in which the caller was working. Used in
// GetIntervalsResponse.
type Interval struct {
	start, end time.Time
}

// GetIntervalsResponse is the result of the GetIntervals calls
type GetIntervalsResponse struct {
	intervals []Interval
}

// Server is the interface exported by the TrackingServer API
type Server interface {
	Tick(req *TickRequest) error
	GetIntervals(req *GetIntervalsRequest) (*GetIntervalsResponse, error)
}

// Implementation

// server implements the Server interface (i.e. the TrackingServer API)
type server struct {
	db tickDB
}

// NewServer returns an implementation of the TrackingServer api
func NewServer() Server {
	return &server{
		db: make(tickDB),
	}
}

// Tick handles the /tick http endpoint
func (s *server) Tick(req *TickRequest) error {
	for _, l := range req.labels {
		s.db[l] = append(s.db[l], time.Now())
	}
	return nil
}

func (s *server) GetIntervals(req *GetIntervalsRequest) (*GetIntervalsResponse, error) {
	// Iterate through 'times' and break it up into intervals
	intervals := []Interval{}
	times := s.db[req.label]
	iL, iR := 0, 0
	for i := 1; i <= len(times); i++ {
		if i == len(times) || times[i].Sub(times[iR]) > 23*time.Minute {
			if iL < iR {
				intervals = append(intervals, Interval{
					start: times[iL],
					end:   times[iR],
				})
			}
			iL = i
		}
		iR = i
	}

	return &GetIntervalsResponse{intervals: intervals}, nil
}
