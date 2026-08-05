package prompt

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// The off-terminal path is the one testable without a pty: Start prints
// one arrow line per phase, Done overtypes it with the final glyph, and
// with the minimal theme the output must contain no escape codes.
func TestProgressGroupOffTTYStaticLines(t *testing.T) {
	var out bytes.Buffer
	pg := NewProgressGroup(&out, GetTheme("minimal", false), []string{"git-init", "readme"})

	pg.Start("generating template go-rest-api")
	pg.Start("applying capability git-init")
	pg.Done("generating template go-rest-api")
	pg.Done("applying capability git-init")
	pg.Start("applying capability readme")
	pg.Done("applying capability readme")

	got := out.String()
	for _, want := range []string{
		"  - generating template go-rest-api",
		"  - applying capability git-init",
		"  [x] generating template go-rest-api",
		"  [x] applying capability git-init",
		"  [x] applying capability readme",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("static output missing %q:\n%q", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("minimal-theme output must not contain escape codes:\n%q", got)
	}
}

// On failure the in-flight phase's arrow line is overtyped with the
// failed glyph; already-done steps keep their done glyph.
func TestProgressGroupOffTTYFail(t *testing.T) {
	var out bytes.Buffer
	pg := NewProgressGroup(&out, GetTheme("minimal", false), []string{"git-init"})

	pg.Start("generating template go-rest-api")
	pg.Done("generating template go-rest-api")
	pg.Start("applying capability git-init")
	pg.Fail()

	got := out.String()
	for _, want := range []string{
		"  [x] generating template go-rest-api",
		"  [X] applying capability git-init",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%q", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("minimal-theme output must not contain escape codes:\n%q", got)
	}
}

// Finish before any Start returns zero and prints nothing.
func TestProgressGroupFinishBeforeStart(t *testing.T) {
	var out bytes.Buffer
	pg := NewProgressGroup(&out, GetTheme("minimal", false), nil)
	if el := pg.Finish(); el != 0 {
		t.Errorf("Finish before any Start: got %v, want 0", el)
	}
	if out.Len() != 0 {
		t.Errorf("Finish before any Start should print nothing, got %q", out.String())
	}
}

// The bar percentage is done/total, never faked: with the template
// phase and one capability (total 2), one completed step is 50%. TTY
// rows carry ANSI color on the glyphs, so assertions target the labels
// and the bar rather than glyph+label byte runs.
func TestProgressGroupTTYStackAndPercent(t *testing.T) {
	var out bytes.Buffer
	pg := NewProgressGroup(&out, GetTheme("default", false), []string{"git-init"})
	pg.tty = true // the byte buffer isn't a terminal; force the TTY path

	pg.Start("generating template go-rest-api")
	pg.Done("generating template go-rest-api")
	pg.Start("applying capability git-init")
	time.Sleep(2 * time.Millisecond)
	elapsed := pg.Finish()

	got := out.String()
	for _, want := range []string{
		"generating template go-rest-api",
		"applying capability git-init",
		"] 50%",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("tty stack missing %q:\n%q", want, got)
		}
	}
	if elapsed <= 0 {
		t.Errorf("Finish elapsed = %v, want > 0", elapsed)
	}
}

// Pending capability rows are reserved up front (visible before their
// phases begin), in selection order. Minimal theme: no escape codes.
func TestProgressGroupTTYPendingRows(t *testing.T) {
	var out bytes.Buffer
	pg := NewProgressGroup(&out, GetTheme("minimal", false), []string{"git-init", "readme"})
	pg.tty = true

	pg.Start("generating template go-rest-api")
	got := out.String()
	if !strings.Contains(got, "- applying capability git-init") || !strings.Contains(got, "- applying capability readme") {
		t.Errorf("pending rows missing:\n%q", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("minimal-theme output must not contain escape codes:\n%q", got)
	}
	pg.Finish()
}

func TestFormatDuration(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0:                       "0.0s",
		920 * time.Millisecond:  "0.9s",
		6200 * time.Millisecond: "6.2s",
		90 * time.Millisecond:   "0.1s",
	} {
		if got := formatDuration(d); got != want {
			t.Errorf("formatDuration(%v) = %q, want %q", d, got, want)
		}
	}
}
