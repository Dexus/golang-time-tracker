package client

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/golang/glog"
	"github.com/msteffen/golang-time-tracker/api"
	"github.com/msteffen/golang-time-tracker/server"
)

var (
	/* const */ dataDir = os.Getenv("HOME") + "/.time-tracker"
	/* const */ dbFile = dataDir + "/db"
	/* const */ socketFile = dataDir + "/sock"
)

// func Today() error {
// 	now := time.Now()
// 	// Make sure to set local time (silly--see https://github.com/msteffen/golang-time-tracker/issues/2)
// 	morning := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
// 	night := morning.Add(24 * time.Hour)
// 	// TODO: don't use hardcoded URLs
// 	url := fmt.Sprintf("http://localhost:10101/intervals?label=%s&start=%d&end=%d",
// 		label, morning.Unix(), night.Unix())
// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return fmt.Errorf("could not create HTTP request for today's intervals: %v", err)
// 	}
// 	resp, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		return fmt.Errorf("could not retrieve today's intervals: %v", err)
// 	}
//
// 	buf := &bytes.Buffer{}
// 	buf.ReadFrom(resp.Body)
// 	var resp GetIntervalsResponse
// 	decoder := json.NewDecoder(buf)
// 	if err := decoder.Decode(&actual); err != nil {
// 		return fmt.Errorf("could not decode response: %v", err)
// 	}
// 	fmt.Printf("%s: ", now.Format("2006/02/01"))
// }

func watchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch <directory>",
		Short: "Start watching the given project directory for writes",
		Long:  "Start watching the given project directory for writes",
		Run: BoundedCommand(1, 1, func(args []string) error {
		}),
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the time-tracker server",
		Long:  "Start the time-tracker server",
		Run: BoundedCommand(1, 1, func(args []string) error {
			flag.Parse() // parse glog flags

			// Set up standard serving dir
			if info, err := os.Stat(dataDir); err != nil {
				if err := os.Mkdir(dataDir, 0755); err != nil {
					return err
				}
			} else if info.Mode().Perm()&0700 != 0700 {
				return fmt.Errorf("must have rwx permissions on %s but only have %s (%0d)",
					DataDir, info.Mode(), info.Mode().Perm()|0700)
			}
			apiServer := api.NewServer(SystemClock, dbFile)
			server.ServeOverHTTP(socketPath, apiServer)
		}),
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Give the status of the time-tracker daemon (or start if it it's stopped)",
		Long:  "Give the status of the time-tracker daemon (or start if it it's stopped)",
		Run: BoundedCommand(1, 1, func(args []string) error {

		}),
	}
}

func main() {
	rootCmd := cobra.Command{
		Use:   "t",
		Short: "T is the client for the golang-time-tracker server",
		Long: "Client-side CLI for a time-tracking/time-gamifying tool that helps " +
			"distractable people use their time more mindfully",
		Run: BoundedCommand(0, 0, func(_ []string) error {
			Today()
		}),
	}
	rootCmd.AddCommand(watchCmd())
	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(statusCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
