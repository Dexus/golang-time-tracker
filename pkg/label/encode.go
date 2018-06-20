// label_encoding.go could almost be a library on its own. It provides a generic
// way to encode sequences of strings as a single string. time-tracker happens
// to use it to encode each tick's labels in single column in SQLite

package label

import (
	"bytes"
	"strings"
)

// EscapeLabel escapes all instances of '"' and '\'
func EscapeLabel(label string) string {
	buf := &bytes.Buffer{}
	// Grow buf to hold the result
	buf.Grow(len(label) +
		strings.Count(label, "\\") +
		strings.Count(label, "\""))
	// Write all labels into 'buf'
	for _, c := range label {
		switch c {
		case '"':
			buf.Write([]byte{'\\', '"'})
		case '\\':
			buf.Write([]byte{'\\', '\\'})
		default:
			buf.WriteRune(c)
		}
	}
	return buf.String()
}

// UnescapeLabel unescapes all instances of '"' and '\'
func UnescapeLabel(in string) string {
	out := &bytes.Buffer{}
	escaped := false
	for _, c := range in {
		if !escaped && c == '\\' {
			escaped = true
			continue
		}
		out.WriteRune(c)
		escaped = false
	}
	return out.String()
}
