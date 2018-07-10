package main

import (
	"testing"
	"time"

	"github.com/msteffen/golang-time-tracker/api"
	tu "github.com/msteffen/golang-time-tracker/testutil"
)

const emptyBarT = "[\033[33m\033[7;33m███████████████████████████████████████████████████████████\033[m]"

func TestEmptyBar(t *testing.T) {
	tu.Check(t, tu.Eq(emptyBar, emptyBarT))
}

func TestBarBasic(t *testing.T) {
	ts := time.Date(
		/* date */ 2017, 7, 1,
		/* time */ 9, 0, 0,
		/* ns ec, lo cation */ 0, time.UTC)
	barStr := Bar(ts, []api.Interval{
		{
			Start: ts.Add(4 * time.Minute).Unix(),
			End:   ts.Add(20 * time.Minute).Unix(),
		},
		{
			Start: ts.Add(60 * time.Minute).Unix(),
			End:   ts.Add(240 * time.Minute).Unix(),
		},
		{
			Start: ts.Add(270 * time.Minute).Unix(),
			End:   ts.Add(306 * time.Minute).Unix(),
		},
	})

	tu.Check(t,
		tu.Eq(barStr,
			"[\033[33m┃\033[7;33m█\033[0;33m▌████████▎\033[7;33m▊███████████████████████████████████████████████]"),
	)
}

func TestBarIntervalFitsInChar(t *testing.T) {
	ts := time.Date(
		/* date */ 2017, 7, 1,
		/* time */ 9, 0, 0,
		/* nsec, location */ 0, time.UTC)
	tu.Check(t,
		tu.Eq(Bar(ts, []api.Interval{
			{
				Start: ts.Add(4 * time.Minute).Unix(),
				End:   ts.Add(20 * time.Minute).Unix(),
			},
		}), "[\033[33m┃\033[7;33m███████████████████████████████████████████████████████████\033[m]"),
	)
}
