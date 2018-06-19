// collector.go is a library for converting ticks into intervals. Ticks are
// "added" to a collector, and when all ticks have been processed, the collector
// is "finished" and all contained intervals are extracted
package main

import (
	"fmt"
	"time"
)

func (i Interval) String() string {
	duration := time.Duration(i.End-i.Start) * time.Second
	start := time.Unix(i.Start, 0)
	return fmt.Sprintf("[%s starting %s (%s)]", duration, start, i.Label)
}

// If this many seconds elapses between consecutive work ticks, then the gap
// will "break" the previous work interval
const maxEventGap int64 = 23 * 60

// Collector is a data structure for converting a sequence of ticks into a
// sequence of intervals (ticks separated by t < maxEventGap)
type Collector struct {
	// lower (left) and upper (right) bound times for all intervals in the
	// collection (overlapping intervals are truncated)
	l, r       int64
	start, end int64 // Start and end time of the 'current' interval (end advances until a 'wide' gap is encountered)
	intervals  []Interval
	label      string
}

// Add adds a tick to 'c'. 's' is the time at which the tick occurred, as a Unix
// timestamp (seconds since epoch)
func (c *Collector) Add(t int64) bool {
	fmt.Printf("Add(%s)", time.Unix(t, 0))
	if c.start > c.r { // no overlap with [l, r]. Nothing to do
		fmt.Println(" - no overlap")
		return false
	} else if t-c.end <= maxEventGap { // Check for interval break
		fmt.Println(" - still going")
		c.end = t // work interval still going: move 'end' to the right
		return true
	}
	fmt.Println(" - interval break")
	c.addInterval()
	c.start, c.end = t, t // start/end of next interval (end will advance)
	return true
}

// Finish indicates that no more ticks will be added. It closes the last
// interval and returns the complete collectiono
func (c *Collector) Finish() []Interval {
	c.addInterval()
	return c.intervals
}

func (c *Collector) addInterval() {
	toAdd := Interval{
		Start: max(c.l, c.start),
		End:   min(c.r, c.end),
		Label: c.label,
	}
	fmt.Printf("%v [max(%s, %s), min(%s, %s)]\n", toAdd, time.Unix(c.l, 0), time.Unix(c.start, 0), time.Unix(c.r, 0), time.Unix(c.end, 0))
	if toAdd.End <= toAdd.Start {
		return // toAdd has duration of 0 (or req.End < toAdd.Start) -- skip
	}
	c.intervals = append(c.intervals, toAdd)
}

func min(l, r int64) int64 {
	if l < r {
		return l
	}
	return r
}

func max(l, r int64) int64 {
	if l > r {
		return l
	}
	return r
}
