package prompt

import (
	"bytes"
	"strings"
	"testing"
)

// The static-line (non-terminal) path is the one testable without a
// pty: Start prints one arrow line per phase, Finish is a no-op, and
// with the minimal theme the output must contain no escape codes.
func TestSpinnerNonTerminalPrintsStaticLines(t *testing.T) {
	var out bytes.Buffer
	sp := NewSpinner(&out, GetTheme("minimal", false))

	sp.Start("generating template go-rest-api")
	sp.Start("applying capability git-init")
	sp.Finish(true)

	got := out.String()
	wantLines := []string{
		"  - generating template go-rest-api",
		"  - applying capability git-init",
	}
	for _, want := range wantLines {
		if !strings.Contains(got, want) {
			t.Errorf("static output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("minimal-theme spinner output must not contain escape codes:\n%q", got)
	}
}

func TestSpinnerNonTerminalFinishIsNoop(t *testing.T) {
	var out bytes.Buffer
	sp := NewSpinner(&out, GetTheme("minimal", false))
	sp.Finish(true)
	if out.Len() != 0 {
		t.Errorf("Finish before any Start should print nothing, got %q", out.String())
	}
}
