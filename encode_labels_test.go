package main

import (
	"testing"
)

func TestEscape(t *testing.T) {
	for _, label := range []string{
		"th\"i\"s", "\"", "\\", "\\is\\", "\"\\\"", "a", "test",
	} {
		Check(t, Eq(unescapeLabel(escapeLabel(label)), label))
	}
}
