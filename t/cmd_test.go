package main

import (
	"testing"
	"time"

	"github.com/msteffen/golang-time-tracker/api"
	tu "github.com/msteffen/golang-time-tracker/testutil"
)

var ts = time.Date(
	/* date */ 2017, 7, 1,
	/* time */ 9, 0, 0,
	/* nsec, location */ 0, time.UTC)

func TestBits(t *testing.T) {
	expected := []byte{
		1, 1, 1, 1, 1, 1, 1, 1,
		4, 4, 4, 4, 4, 4,
		6, 7, 7, 7, 8,
	}
	for i, c := range []byte{
		0x01, 0x02, 0x04, 0x08, 0x10, 0x20, 0x40, 0x80,
		0x55, 0xaa, 0x33, 0xcc, 0x0f, 0xf0,
		0xdd, 0xdf, 0xef, 0xfe, 0xff,
	} {
		tu.Check(t, tu.Eq(bits(c), expected[i]))
	}
}

func TestEmptyBar(t *testing.T) {
	barStr := Bar(ts, []api.Interval{})
	tu.Check(t, tu.Eq(barStr,
		"[\x1b[7;33m████████████████████████████████████████████████████████████\x1b[m]"))
}

func TestBarBasic(t *testing.T) {
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
		tu.HasPrefix(
			barStr,
			"[\x1b[33m┃\x1b[7;33m█▌\x1b[0;33m███████\x1b[7;33m█▎\x1b[0;33m▊\x1b[7;33m███"),
		tu.HasSuffix(barStr, "█████████████████████████████\x1b[m]"),
	)
}

func TestBarIntervalFitsInChar(t *testing.T) {
	barStr := Bar(ts, []api.Interval{
		{
			Start: ts.Add(4 * time.Minute).Unix(),
			End:   ts.Add(20 * time.Minute).Unix(),
		},
	})
	tu.Check(t,
		tu.HasPrefix(barStr, "[\x1b[33m┃\x1b[7;33m███"),
		tu.HasSuffix(barStr, "████████████████████████████████████████████████████████\x1b[m]"),
	)
}
