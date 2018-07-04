package testutil

import (
	"reflect"
	"testing"
)

// Cond is a generic wrapper around a test check. Conds are generally created
// with Eq, Nil, etc. For example:
// Check(
//   Nil(err),
//   Eq(result, 7),
// )
type Cond struct {
	// Ok is true if the condition being checked (e.g that a == b in the case of
	// Eq(a, b)) and false otherwise
	Ok bool

	// Msg is a formatting message to display if the condition is violated
	Msg string

	// Args are any arguments to be placed into the format string if the condition
	// is violated
	Args []interface{}
}

// Nil confirms that 'err' is nil, and calls t.Fatal() otherwise
func Nil(err error) Cond {
	return Cond{
		Ok: err == nil,
		Msg: "expected <nil> error, but was:\n	%v",
		Args: []interface{}{err},
	}
}

// Eq confirms that 'expected' and 'actual' are equal, and calls t.Fatal()
// otherwise
func Eq(actual interface{}, expected interface{}) Cond {
	return Cond{
		Ok:   reflect.DeepEqual(expected, actual),
		Msg:  "expected: \"%+v\"\nbut was: \"%+v\"",
		Args: []interface{}{expected, actual},
	}
}

// Check checks one or more testing conditions
func Check(t *testing.T, conds ...Cond) {
	t.Helper()
	for _, cond := range conds {
		if !cond.Ok {
			t.Fatalf(cond.Msg, cond.Args...)
		}
	}

}
