package testutil

import (
	"reflect"
	"testing"

	"github.com/msteffen/golang-time-tracker/api"
)

var intervalRespType = reflect.ValueOf(api.GetIntervalsResponse{}).Type()

// Check assumptions that this library makes about the structure of
// GetIntervalsResponse on startup
func init() {
	if intervalRespType.NumField() != 1 {
		panic("GetIntervalsResponse now has more than one field; need to update Eq")
	}
}

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
	ok := false

	expectedVal := reflect.ValueOf(expected)
	actualVal := reflect.ValueOf(actual)
	switch {
	case expectedVal.Kind() == reflect.Slice &&
		actualVal.Kind() == reflect.Slice &&
		expectedVal.Len() == 0 && actualVal.Len() == 0:
		// handle nil slice vs empty slice
		ok = true
	case expectedVal.Type() == intervalRespType &&
		actualVal.Type() == intervalRespType:
		// Handle GetIntervalResponses with nil vs empty .Intervals
		return Eq(actualVal.FieldByName("Intervals").Interface(),
			expectedVal.FieldByName("Intervals").Interface())
	default:
		// Handle all other cases
		ok = reflect.DeepEqual(expected, actual)
	}
	return Cond{
		Ok:   ok,
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
