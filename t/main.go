package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/msteffen/golang-time-tracker/api"
	cu "github.com/msteffen/golang-time-tracker/clientutil"
	"github.com/msteffen/golang-time-tracker/server"
)

var (
	/* const */ dataDir = os.Getenv("HOME") + "/.time-tracker"
	/* const */ dbFile = dataDir + "/db"
	/* const */ socketFile = dataDir + "/sock"
)

func Today() error {
	now := time.Now()
	// Make sure to set local time (silly--see https://github.com/msteffen/golang-time-tracker/issues/2)
	morning := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	night := morning.Add(24 * time.Hour)
	c := cu.GetClient(socketFile)
	httpResp, err := c.Get(fmt.Sprintf("/intervals?start=%d&end=%d", morning.Unix(), night.Unix()))
	if err != nil {
		return fmt.Errorf("could not retrieve today's intervals: %v", err)
	}
	var resp api.GetIntervalsResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return fmt.Errorf("could not decode response: %v", err)
	}
	// block chars = u2588 (full) - u258f (left eighth)
	fmt.Printf("%s: %s", morning.Format("2006/02/01 "), Bar(morning, resp.Intervals))
	return nil
}

// func watchCmd() *cobra.Command {
// 	return &cobra.Command{
// 		Use:   "watch <directory>",
// 		Short: "Start watching the given project directory for writes",
// 		Long:  "Start watching the given project directory for writes",
// 		Run: BoundedCommand(1, 1, func(args []string) error {
// 		}),
// 	}
// }

func tickCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tick <label>",
		Short: "Append a tick (work event) with the given label",
		Long:  "Append a tick (work event) with the given label",
		Run: BoundedCommand(1, 1, func(args []string) error {
			c := cu.GetClient(socketFile)
			resp, err := c.PostString("/tick", fmt.Sprintf("{ label: \"%s\" }", args[0]))
			if err != nil {
				buf := &bytes.Buffer{}
				io.Copy(buf, resp.Body)
				return fmt.Errorf("%s (%v)", buf.String(), err)
			}
			return nil
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
				return fmt.Errorf("must have rwx permissions on %s but only have %s (%0d vs 0700)",
					dataDir, info.Mode(), info.Mode().Perm()&0700)
			}
			apiServer, err := api.NewServer(api.SystemClock, dbFile)
			if err != nil {
				return fmt.Errorf("could not create APIServer: %v", err)
			}
			return server.ServeOverHTTP(socketFile, api.SystemClock, apiServer)
		}),
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Give the status of the time-tracker daemon (or start if it it's stopped)",
		Long:  "Give the status of the time-tracker daemon (or start if it it's stopped)",
		Run: BoundedCommand(1, 1, func(args []string) error {
			c := cu.GetClient(socketFile)
			resp, err := c.Get("/status")
			if err != nil {
				return fmt.Errorf("GET error: %v", err)
			}
			buf := &bytes.Buffer{}
			io.Copy(buf, resp.Body)
			fmt.Printf("time-tracker has been up for %s", buf.String())
			return nil
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
			// Today()
			return nil
		}),
	}
	// rootCmd.AddCommand(watchCmd())
	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(tickCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
