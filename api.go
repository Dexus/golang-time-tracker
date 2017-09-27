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
	Labels []string
}

// GetIntervalsRequest is the object sent to the /get-intervals endpoint.
type GetIntervalsRequest struct {
	// The label whose intervals we want to get
	Label string
}

// Interval represents a time interval in which the caller was working. Used in
// GetIntervalsResponse.
type Interval struct {
	Start, End time.Time
}

// GetIntervalsResponse is the result of the GetIntervals calls
type GetIntervalsResponse struct {
	Intervals []Interval
}

// Server is the interface exported by the TrackingServer API
type Server interface {
	Tick(req *TickRequest) error
	GetIntervals(req *GetIntervalsRequest) (*GetIntervalsResponse, error)
}

// Implementation

// server implements the Server interface (i.e. the TrackingServer API)
type server struct {
	db    tickDB
	clock Clock
}

// NewServer returns an implementation of the TrackingServer api
func NewServer(c Clock) Server {
	return &server{
		db:    make(tickDB),
		clock: c,
	}
}

// Tick handles the /tick http endpoint
func (s *server) Tick(req *TickRequest) error {
	for _, l := range req.Labels {
		s.db[l] = append(s.db[l], s.clock.Now())
	}
	return nil
}

func (s *server) GetIntervals(req *GetIntervalsRequest) (*GetIntervalsResponse, error) {
	// Iterate through 'times' and break it up into intervals
	intervals := []Interval{}
	times := s.db[req.Label]
	iL, iR := 0, 0
	for i := 1; i <= len(times); i++ {
		if i == len(times) || times[i].Sub(times[iR]) > 23*time.Minute {
			if iL < iR {
				intervals = append(intervals, Interval{
					Start: times[iL],
					End:   times[iR],
				})
			}
			iL = i
		}
		iR = i
	}

	return &GetIntervalsResponse{Intervals: intervals}, nil
}
