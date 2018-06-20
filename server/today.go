package main

import (
	"html/template"
	"net/http"
	"time"
)

type div struct {
	Left, Width int
}

// TodayOp has all of the internal data structures retrieved/computed while
// generating the /today page
type TodayOp struct {
	//// Not Owned
	// The 'server' that handles incoming requests (parent struct; owns this
	// TodayOp)
	server APIServer
	// The clock used by 'server' for testing
	clock Clock
	// The http response writer that must receive the result of /today
	writer http.ResponseWriter

	//// Owned
	// the set of intervals we request from 'server' and must render
	intervals []Interval
	// The intervals in 'intervals' converted to an IR that is easy to render
	divs []div
	// The width of the result html page's background
	bgWidth float64
}

func (t *TodayOp) start() {
	t.getIntervals()
}

// getIntervals generates 'div' structs indicating where "work" divs should be
// placed (which indicate time when I was working)
func (t *TodayOp) getIntervals() {
	now := t.clock.Now()
	morning := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	result, err := t.server.GetIntervals(&GetIntervalsRequest{
		Start: morning.Unix(),
		End:   morning.Add(24 * time.Hour).Unix(),
		Label: "",
	})
	if err != nil {
		http.Error(t.writer, err.Error(), http.StatusInternalServerError)
		return
	}
	t.intervals = result.Intervals
	t.computeDivs()
}

func (t *TodayOp) computeDivs() {
	morning := func() int64 {
		now := t.clock.Now()
		m := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return m.Unix()
	}()
	daySecs := (24 * time.Hour).Seconds()
	t.divs = make([]div, 0, len(t.intervals))
	for _, i := range t.intervals {
		t.divs = append(t.divs, div{
			Left:  int(t.bgWidth * float64(i.Start-morning) / daySecs),
			Width: int(t.bgWidth * float64(i.End-i.Start) / daySecs),
		})
	}
	t.generateTemplate()
}

func (t *TodayOp) generateTemplate() {
	// Place generated divs into HTML template
	data, err := Asset(`today.html.template`)
	if err != nil {
		http.Error(t.writer, "could not load today.html.template: "+err.Error(),
			http.StatusInternalServerError)
	}
	err = template.Must(template.New("").Funcs(template.FuncMap{
		"bgWidth": func() int { return int(t.bgWidth) },
	}).Parse(string(data))).Execute(t.writer, t.divs)
	if err != nil {
		http.Error(t.writer, err.Error(), http.StatusInternalServerError)
		return
	}
}
