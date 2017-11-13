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
	// The 'server' that handles incoming requests (parent struct; owns this)
	server *server
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
	result, err := t.server.GetIntervals(&GetIntervalsRequest{
		// Start: <FILL IN>,
		// End: <FILL IN>,
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
	now := t.server.clock.Now()
	morning := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	daySecs := (24 * time.Hour).Seconds()
	divs := make([]div, 0, len(t.intervals))
	for _, i := range t.intervals {
		divs = append(divs, div{
			Left:  int((t.bgWidth * i.Start.Sub(morning).Seconds()) / daySecs),
			Width: int((t.bgWidth * i.End.Sub(i.Start).Seconds()) / daySecs),
		})
	}
	t.generateTemplate()
}

func (t *TodayOp) generateTemplate() {
	// Place generated divs into HTML template
	err := template.Must(template.New("").Funcs(template.FuncMap{
		"bgWidth": func() int { return int(t.bgWidth) },
	}).Parse(`
		<head>
			<style type="text/css">
				.timebg {
					width: {{bgWidth}}pt;
					height: 60pt;
					top: 20pt;
					margin: auto;
					background-color: #d5d5d5;
				}
				.timefg {
					height: 60pt;
					background-color: #ffb915;
					display: inline-block;
				}
			</style>
		</head>
		<body>
		<div class="timebg">
		{{range .}}
			<div class="timefg" style="position: relative; left: {{.Left}}pt; width: {{.Width}}pt;">
			</div>
		{{end}}
		</div>
		</body>
		`)).Execute(t.writer, t.divs)
	if err != nil {
		http.Error(t.writer, err.Error(), http.StatusInternalServerError)
		return
	}
}
