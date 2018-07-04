package api

import (
	"testing"

	tu "github.com/msteffen/golang-time-tracker/pkg/testutil"
)

func TestBasic(t *testing.T) {
	c := &Collector{
		l: 0,
		r: 24 * 60 * 60,
	}
	curT := int64(0)
	c.Add(curT)
	for _, delta := range []int64{
		1, 1, 1, 1, 1,
		23*60 + 1, 1, 1, 1, 1, 1,
	} {
		curT += delta
		c.Add(curT)
	}
	c.Finish()
	tu.Check(t, tu.Eq(c.intervals, []Interval{
		{Start: 0, End: 5},
		{Start: 23*60 + 6, End: 23*60 + 11},
	}))
}
