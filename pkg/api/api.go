package api

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"math"
	"sync"
	"time"
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

	// The label whose intervals we want to get. If the label is the empty string
	// or unset, then get all intervals
	Label string
}

// Interval represents a time interval in which the caller was working. Used in
// GetIntervalsResponse.
type Interval struct {
	Start, End int64 // start and end times, as int64 seconds since epoch
	Label      string
}

// GetIntervalsResponse is the result of the GetIntervals calls
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
func NewServer(c Clock, dbFile string) (APIServer, error) {
	// Create DB connectin
	db, err := sql.Open("sqlite3", dbFile)
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
		clock: c,
	}, nil
}

// Tick handles the /tick http endpoint
func (s *server) Tick(req *TickRequest) error {
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
		if req.Label == "" {
			rows, err = s.db.Query(fmt.Sprintf(
				"SELECT * FROM ticks WHERE time BETWEEN %d AND %d", start, end,
			))
		} else {
			rows, err = s.db.Query(fmt.Sprintf(
				"SELECT * FROM ticks WHERE time BETWEEN %d AND %d AND labels LIKE \"%%%s%%\"",
				start, end, EscapeLabel(req.Label),
			))
		}
	}()
	if err != nil {
		return nil, err
	}

	// Iterate through 'times' and break it up into intervals
	collector := make(map[string]*Collector) // map label to collector
	for rows.Next() {
		// parse SQL record
		var escapedLabel string
		var t int64
		rows.Scan(&t, &escapedLabel)
		fmt.Printf("%s, %s\n", time.Unix(t, 0), escapedLabel)
		label := UnescapeLabel(escapedLabel)
		if req.Label != "" && req.Label != label {
			continue
		}
		if collector[label] == nil {
			collector[label] = &Collector{
				l:     req.Start,
				r:     req.End,
				label: label,
			}
		}
		collector[label].Add(t)
	}

	// finish collectors
	collections := make([][]Interval, 0, len(collector)) // each label's intervals
	sz := 0
	for _, c := range collector {
		collections = append(collections, c.Finish())
		sz += len(collections[len(collections)-1])
	}

	// merge intervals into sorted list
	intervals := make([]Interval, 0, sz) // final list of intervals to return
	for {
		// remove empty collections (copy from -> to, where 'from' skips empties)
		to := 0
		for from := 0; from < len(collections); from++ {
			if len(collections[from]) == 0 { // from is empty -- skip
				continue
			}
			collections[to] = collections[from]
			to++
		}
		collections = collections[:to]
		if len(collections) == 0 {
			break // all empty
		}

		// scan through first element of all non-empty collections and find the min
		tmin := int64(math.MaxInt64) // math.MaxInt64 is const, not an int64, sadly
		imin := len(collections)
		for i := len(collections) - 1; i >= 0; i-- {
			// not all collections are empty => not done
			if collections[i][0].Start < tmin {
				tmin = collections[i][0].Start
				imin = i
			}
		}
		// add min interval to 'intervals'
		intervals = append(intervals, collections[imin][0])
		if len(collections[imin]) > 1 {
			collections[imin] = collections[imin][1:]
		} else {
			collections[imin] = nil
		}
	}

	return &GetIntervalsResponse{Intervals: intervals}, nil
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
