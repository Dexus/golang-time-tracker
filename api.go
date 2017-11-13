package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// If this many minutes elapses between consecutive work ticks, then the gap
// will "break" the previous work interval
const maxEventGap = 23 * time.Minute

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
	// The time period in which we want to get intervals. If an interval in the
	// result overlaps with 'Start' or 'End', it will be truncated.
	Start, End time.Time

	// The label whose intervals we want to get. If the label is the empty string
	// or unset, then get all intervals
	Label string
}

// Interval represents a time interval in which the caller was working. Used in
// GetIntervalsResponse.
type Interval struct {
	Start, End time.Time
	Labels     []string
}

func (i Interval) String() string {
	return fmt.Sprintf("[%s starting %s]", i.End.Sub(i.Start), i.Start)
}

// GetIntervalsResponse is the result of the GetIntervals calls
type GetIntervalsResponse struct {
	Intervals []Interval
}

// Server is the interface exported by the TrackingServer API
type Server interface {
	Tick(req *TickRequest) error
	GetIntervals(req *GetIntervalsRequest) (*GetIntervalsResponse, error)
	GetToday(w http.ResponseWriter)
	Clear()
}

// Implementation

// server implements the Server interface (i.e. the TrackingServer API)
type server struct {
	//// Not owned
	clock Clock

	//// Owned
	mu sync.Mutex
	db tickDB
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
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, l := range req.Labels {
		fmt.Printf("time: %v\n", s.clock.Now())
		s.db[l] = append(s.db[l], s.clock.Now())
		s.db[""] = append(s.db[""], s.clock.Now())
	}
	return nil
}

func min(t1 time.Time, t2 time.Time) time.Time {
	if t1.Before(t2) {
		return t1
	}
	return t2
}

func max(t1 time.Time, t2 time.Time) time.Time {
	if t1.After(t2) {
		return t1
	}
	return t2
}

func (s *server) GetIntervals(req *GetIntervalsRequest) (*GetIntervalsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Iterate through 'times' and break it up into intervals
	intervals := []Interval{}
	times := s.db[req.Label]
	var iL int // left (lower) bound of interval - always starts at 0
	for i := 1; i <= len(times) && times[iL].Before(req.End); i++ {
		iR := i - 1 // potential right (upper) bound of interval
		if i < len(times) &&
			times[iR].Before(req.End) &&
			times[i].Sub(times[iR]) <= maxEventGap {
			continue // work interval still going -- move iR to the right
		}

		// Interval break between i-1 (iR) and i
		// Add prev interval to toAdd and advance iL to start a new interval
		// (next iteration)
		toAdd := Interval{
			Start: max(times[iL], req.Start),
			End:   min(times[iR], req.End),
		}
		iL = i
		if toAdd.End.Sub(toAdd.Start) <= 0 {
			continue // toAdd has duration of 0 (or req.End < toAdd.Start) -- skip
		}
		if times[iR].Before(req.Start) {
			continue // toAdd doesn't overlap with request -- skip
		}
		intervals = append(intervals, toAdd)
	}
	return &GetIntervalsResponse{Intervals: intervals}, nil
}

// GetToday writes the http response for the /today page to 'w'.
func (s *server) GetToday(w http.ResponseWriter) {
	t := TodayOp{
		server:  s,
		writer:  w,
		bgWidth: float64(500),
	}
	t.start()
}

func (s *server) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db = make(tickDB)
}
