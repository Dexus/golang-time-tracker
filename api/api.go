package api

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"sync"
	"time"

	"github.com/golang/glog"
)

// -------------- API --------------

// TickRequest is an object sent to the /tick http endpoint, to indicate a file
// save or some other task-related action has occurred
type TickRequest struct {
	// The label (i.e. task) on which the user is currently working
	Label string
}

// GetIntervalsRequest is the object sent to the /get-intervals endpoint.
type GetIntervalsRequest struct {
	// The time period in which we want to get intervals, as seconds since epoch.
	// If an interval in the result overlaps with 'Start' or 'End', it will be
	// truncated.
	Start, End int64
}

// Interval represents a time interval in which the caller was working. Used in
// GetIntervalsResponse.
type Interval struct {
	Start, End int64 // start and end times, as int64 seconds since epoch

	// The activity that was done in this interval (or "" if multiple activities
	// may have occurred)
	Label string
}

// GetIntervalsResponse contains all activity intervals, clamped to the
// requested start/end times, sorted by start time
type GetIntervalsResponse struct {
	Intervals []Interval
}

// APIServer is the interface exported by the TrackingServer API
type APIServer interface {
	Tick(req *TickRequest) error
	GetIntervals(req *GetIntervalsRequest) (*GetIntervalsResponse, error)
	Clear() error
}

// --------- Implementation --------

// server implements the Server interface (i.e. the TrackingServer API)
type server struct {
	//// Not owned
	clock Clock

	//// Owned
	db *sql.DB
	// The sqlite driver does not allow for concurrent writes. See
	// https://github.com/mattn/go-sqlite3#faq
	// This allows for safe concurrent use of 'db'
	mu sync.RWMutex
}

// NewServer returns an implementation of the TrackingServer api
func NewServer(clock Clock, dbPath string) (APIServer, error) {
	// Create DB connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	for err != nil {
		time.Sleep(time.Second)
		err = db.Ping()
	}
	// Take advantage of sqlite INTEGER PRIMARY KEY table for fast range scan of
	// ticks: https://sqlite.org/lang_createtable.html#rowid
	if _, err = db.Exec(
		`CREATE TABLE IF NOT EXISTS ticks (time INTEGER PRIMARY KEY ASC, labels TEXT)`,
	); err != nil {
		return nil, err
	}
	return &server{
		db:    db,
		clock: clock,
	}, nil
}

// Tick handles the /tick http endpoint
func (s *server) Tick(req *TickRequest) error {
	// Validate req
	if req.Label == "" {
		return fmt.Errorf("tick request must have a label (\"\" is used to " +
			"indicate intervals formed by the union of all ticks in GetIntervals")
	}

	// Write tick to DB
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(fmt.Sprintf(
		"INSERT INTO ticks VALUES (%d, \"%s\")", s.clock.Now().Unix(), EscapeLabel(req.Label),
	))
	return err
}

func (s *server) GetIntervals(req *GetIntervalsRequest) (*GetIntervalsResponse, error) {
	// Get list of times in the 'req' range from DB
	var rows *sql.Rows
	var err error
	func() {
		s.mu.RLock()
		defer s.mu.RUnlock()
		// check maxEventGap before and after request, to handle the case where a time
		// interval overlaps with the request interval
		start := req.Start - maxEventGap
		end := req.End + maxEventGap
		rows, err = s.db.Query(fmt.Sprintf(
			"SELECT * FROM ticks WHERE time BETWEEN %d AND %d", start, end,
		))
	}()
	if err != nil {
		return nil, err
	}

	// Iterate through 'times' and break it up into intervals
	collector := make(map[string]*Collector) // map label to collector
	collector[""] = &Collector{
		l: req.Start,
		r: req.End,
	}
	var (
		prevLabel string // label that no tick will have initially
		prevT     int64  // prev tick's time (unix seconds)
	)
	for rows.Next() {
		// parse SQL record
		var escapedLabel string
		var t int64
		rows.Scan(&t, &escapedLabel)
		glog.Infof("%s, %s\n", time.Unix(t, 0), escapedLabel)
		label := UnescapeLabel(escapedLabel)

		// Add timestamp to collectors
		if collector[label] == nil {
			collector[label] = &Collector{
				l:     req.Start,
				r:     req.End,
				label: label,
			}
		}
		// this activity's interval starts at the end of the previous activity's
		// interval (if there is one)
		if prevLabel != label {
			if prevT > 0 {
				collector[label].Add(t)
			}
			prevLabel = label
			prevT = t
		}
		collector[label].Add(t)
		collector[""].Add(t)
	}

	// TODO include labelled intervals in response
	return &GetIntervalsResponse{Intervals: collector[""].Finish()}, nil
}

func (s *server) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.db.Exec(`
	  DROP TABLE ticks;
	  CREATE TABLE IF NOT EXISTS ticks (time INTEGER PRIMARY KEY ASC, labels TEXT);
	`); err != nil {
		return err
	}
	return nil
}
