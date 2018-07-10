package testutil

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
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
}

// Nil confirms that 'err' is nil, and calls t.Fatal() otherwise
func Nil(err error) Cond {
	return Cond{
		Ok: err == nil,
		Msg: fmt.Sprintf("expected <nil> error, but was:\n	%v", err),
	}
}

// HasPrefix confirms that 'text' has the prefix 'prefix', and calls t.Fatal()
// otherwise
func HasPrefix(text interface{}, prefix interface{}) Cond {
	cStr, cOK := text.(string)
	pStr, pOK := prefix.(string)
	if cOK && pOK {
		return Cond{
			Ok:  strings.HasPrefix(cStr, pStr),
			Msg: fmt.Sprintf("expected: %q\n to be prefix of: %q\nbut it was not", pStr, cStr),
		}
	}
	cBytes, cOK := text.([]byte)
	pBytes, pOK := prefix.([]byte)
	if cOK && pOK {
		return Cond{
			Ok:  bytes.HasPrefix(cBytes, pBytes),
			Msg: fmt.Sprintf("expected: %q\n to be prefix of: %q\nbut it was not", pBytes, cBytes),
		}
	}
	return Cond{
		Ok:  false,
		Msg: fmt.Sprintf("expected (string, string) or ([]byte, []byte) but got (%T, %T)", text, prefix),
	}
}

// HasSuffix confirms that 'text' has the suffix 'suffix', and calls t.Fatal()
// otherwise
func HasSuffix(text interface{}, suffix interface{}) Cond {
	cStr, cOK := text.(string)
	pStr, pOK := suffix.(string)
	if cOK && pOK {
		return Cond{
			Ok:  strings.HasSuffix(cStr, pStr),
			Msg: fmt.Sprintf("expected: %q\n to be suffix of: %q\nbut it was not", pStr, cStr),
		}
	}
	cBytes, cOK := text.([]byte)
	pBytes, pOK := suffix.([]byte)
	if cOK && pOK {
		return Cond{
			Ok:  bytes.HasSuffix(cBytes, pBytes),
			Msg: fmt.Sprintf("expected: %q\n to be suffix of: %q\nbut it was not", pBytes, cBytes),
		}
	}
	return Cond{
		Ok:  false,
		Msg: fmt.Sprintf("expected (string, string) or ([]byte, []byte) but got (%T, %T)", text, suffix),
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
	// Quote strings for easier debugging
	if e, ok := expected.(string); ok {
		expected = interface{}(fmt.Sprintf("%q", e)[1 : len(e)+1])
	}
	if a, ok := actual.(string); ok {
		actual = interface{}(fmt.Sprintf("%q", actual)[1 : len(a)+1])
	}
	return Cond{
		Ok:  ok,
		Msg: fmt.Sprintf("expected: \"%+v\"\n         but was: \"%+v\"", expected, actual),
	}
}

// Check checks one or more testing conditions
func Check(t *testing.T, conds ...Cond) {
	t.Helper()
	for _, cond := range conds {
		if !cond.Ok {
			t.Fatal(cond.Msg)
		}
	}

}
