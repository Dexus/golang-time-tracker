package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
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
			http.Error(w, "request did not match expected type", http.StatusBadRequest)
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

		// Process request
		req := GetIntervalsRequest{Label: r.URL.Query().Get("label")} // GET param
		result, err := s.GetIntervals(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resultJSON, err := json.Marshal(result)
		if err != nil {
			http.Error(w, "could not serialize result: "+err.Error(), http.StatusInternalServerError)
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

		req := make(map[string]interface{})
		d := json.NewDecoder(r.Body)
		if err := d.Decode(req); err != nil {
			http.Error(w, "request did not match expected type", http.StatusBadRequest)
			return
		}
		if req["confirm"] != "yes" {
			http.Error(w, "Must send confirmation message to delete all server data", http.StatusBadRequest)
		}
		s.Clear()
		w.WriteHeader(http.StatusOK)
	})
}

func handleToday(c Clock, s Server) {
	http.HandleFunc("/today", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("handling /today")
		// Unmarshal and validate request
		if r.Method != "GET" {
			http.Error(w, "must use GET to access /today", http.StatusMethodNotAllowed)
			return
		}

		// Generate 'div' structs indicating where "work" divs should be placed
		// (which indicate time when I was working)
		result, err := s.GetIntervals(&GetIntervalsRequest{Label: ""})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		type div struct {
			Left, Width int
		}
		now := c.Now()
		morning := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		daySecs := (24 * time.Hour).Seconds()
		divs := make([]div, 0, len(result.Intervals))
		bgWidth := float64(500)
		for _, i := range result.Intervals {
			divs = append(divs, div{
				Left:  int((bgWidth * i.Start.Sub(morning).Seconds()) / daySecs),
				Width: int((bgWidth * i.End.Sub(i.Start).Seconds()) / daySecs),
			})
		}

		// Place generated divs into HTML template
		err = template.Must(template.New("").Funcs(template.FuncMap{
			"bgWidth": func() int { return int(bgWidth) },
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
		`)).Execute(w, divs)
		if err != nil {
			w.Write([]byte(err.Error()))
		}
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

func startServing(c Clock) {
	s := NewServer(c)
	handleTick(s)
	handleGetIntervals(s)
	handleToday(c, s)

	// Return to non-endpoint calls with 404
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	// Start serving requests
	log.Fatal(http.ListenAndServe(":10101", nil))
}

func main() {
	startServing(SystemClock{})
}
