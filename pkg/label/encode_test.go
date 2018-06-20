package label

import (
	"testing"

	tu "github.com/msteffen/golang-time-tracker/pkg/testutil"
)

func TestEscape(t *testing.T) {
	for _, label := range []string{
		"th\"i\"s", "\"", "\\", "\\is\\", "\"\\\"", "a", "test",
	} {
		tu.Check(t, tu.Eq(UnescapeLabel(EscapeLabel(label)), label))
	}
}
