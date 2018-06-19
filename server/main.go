package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"time"
)

var (
	/* const */ dataDir = os.Getenv("HOME") + "/.time-tracker"
	/* const */ dbFile = dataDir + "/ticks.db"
	/* const */ pidFile = dataDir + "/pid"
)

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

func startServing(c Clock, file string) {
	if file == "" {
		if err := func() error {
			// Create data dir if it doesn't exist
			if _, err := os.Stat(dataDir); err == os.ErrNotExist {
				if err := os.Mkdir(dataDir, 0644); err != nil {
					return err
				}
			}
			// Create a pid file to make sure we're not starting a redundant server (or
			// error if one exists)
			if f, err := os.OpenFile(pidFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0544); err != os.ErrExist {
				f.Write(append(strconv.AppendInt(nil, int64(os.Getpid()), 10), '\n'))
				f.Close()
				return nil // success: no other servers, start as usual
			}
			f, err := os.Open(pidFile)
			if err != nil {
				return fmt.Errorf("pid file exists at %s, however it can't be opened; "+
					"refusing to start to avoid DB corruption", pidFile)
			}
			pid, port := "", ""
			s := bufio.NewScanner(f)
			if s.Scan() {
				pid = s.Text()
			}
			if s.Scan() {
				port = s.Text()
			}
			if _, err := os.Stat("/proc/" + pid); os.IsNotExist(err) {
				// pidfile points at proc that has died. Delete pidfile and try again
				os.Remove(pidFile)
				startServing(c, file)
				return nil // doesn't matter -- this never returns
			}
			// TODO this is awfully complicated -- am I ever going to use a non-default
			// port?
			switch {
			case pid != "" && port != "":
				return fmt.Errorf("time-tracker server already running at pid %s "+
					"(port %s)", pid, port)
			case pid != "":
				return fmt.Errorf("time-tracker server already running at pid %s",
					pid)
			default:
				return fmt.Errorf("pid file exists at %s, however it's empty (it "+
					"may have content in a moment); refusing to start to avoid DB "+
					"corruption", pidFile)

			}
		}(); err != nil {
			fmt.Fprintf(os.Stderr, "Could not check for existing server: %v", err)
			os.Exit(1)
		}
	}
	s, err := NewServer(c, file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not start server: %v", err)
		os.Exit(1)
	}
	ServeOverHTTP(s, c)
}

func main() {
	startServing(SystemClock{}, "" /* default db file */)
}
