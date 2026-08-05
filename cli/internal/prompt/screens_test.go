package prompt

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/intruder0007/Lumo/core/engine"
	"github.com/intruder0007/Lumo/core/plugin"
)

// The failure screen never exposes the plugin binary's on-disk path —
// both the bare form ("starting C:\…\go-rest-api:") and the
// strconv.Quote'd form inside the wrapped exec error ("C:\\…") are
// scrubbed to <binary>.
func TestErrorMessageScrubsEntrypointPath(t *testing.T) {
	path := `C:\code\plugins\go-rest-api\go-rest-api`
	start := &plugin.StartError{
		EntrypointPath: path,
		Err:            errors.New(`exec: "C:\\code\\plugins\\go-rest-api\\go-rest-api": executable file not found in %PATH%`),
	}
	err := fmt.Errorf("generating from template %q: %w", "go-rest-api", start)

	got := errorMessage(err)
	if strings.Contains(got, `C:\code`) {
		t.Errorf("errorMessage leaked the entrypoint path:\n%q", got)
	}
	if !strings.Contains(got, "<binary>") {
		t.Errorf("errorMessage should name the binary as <binary>:\n%q", got)
	}
	if !strings.Contains(got, "generating from template \"go-rest-api\"") {
		t.Errorf("errorMessage should keep the outer context:\n%q", got)
	}
}

// Non-plugin errors pass through untouched — the failure screen's job is
// scrubbing binary paths, not rewriting every message.
func TestErrorMessagePassesThroughOtherErrors(t *testing.T) {
	err := errors.New("target directory /tmp/x already exists and is not empty")
	if got := errorMessage(err); got != err.Error() {
		t.Errorf("errorMessage altered a non-plugin error: %q", got)
	}
}

// NO_COLOR/--no-color forces the ASCII rendition of every glyph, so no
// state exists only as a unicode icon even on the icon theme.
func TestNoColorGlyphFallback(t *testing.T) {
	plain := GetTheme("default", true)
	if plain.UseColor {
		t.Fatal("no-color default theme must have UseColor=false")
	}
	for kind, want := range map[GlyphKind]string{
		GlyphCursor:  ">",
		GlyphChecked: "[x]",
		GlyphDone:    "[x]",
		GlyphFailed:  "[X]",
		GlyphPending: "-",
		GlyphArrow:   "-",
		GlyphCaret:   "_",
	} {
		if got := plain.Glyph(kind); got != want {
			t.Errorf("no-color Glyph(%d) = %q, want %q", kind, got, want)
		}
	}

	colored := GetTheme("default", false)
	if got := colored.Glyph(GlyphDone); got != "✔" {
		t.Errorf("default-theme Glyph(Done) = %q, want ✔", got)
	}
}

// The minimal theme's glyphs are ASCII to begin with; GetTheme must not
// change them under noColor.
func TestMinimalThemeGlyphsAreStable(t *testing.T) {
	got := GetTheme("minimal", true)
	for kind, want := range map[GlyphKind]string{
		GlyphCursor:    ">",
		GlyphChecked:   "[x]",
		GlyphUnchecked: "[ ]",
		GlyphDone:      "[x]",
		GlyphFailed:    "[X]",
		GlyphArrow:     "-",
	} {
		if g := got.Glyph(kind); g != want {
			t.Errorf("minimal Glyph(%d) = %q, want %q", kind, g, want)
		}
	}
}

// ErrorPanel (O-04 Failure, screen-spec S09) is unbordered: a headline,
// an optional hint, an optional reference, each on its own line, framed
// by a leading and trailing blank line.
func TestErrorPanelShape(t *testing.T) {
	theme := GetTheme("minimal", false)

	full := ErrorPanel(theme, "boom", "hint: try again", "docs/foo.md")
	want := []string{"", "[FAIL] boom", "  hint: try again", "  reference: docs/foo.md", ""}
	if len(full) != len(want) {
		t.Fatalf("ErrorPanel with hint+ref: got %d lines, want %d: %q", len(full), len(want), full)
	}
	for i := range want {
		if full[i] != want[i] {
			t.Errorf("ErrorPanel line %d = %q, want %q", i, full[i], want[i])
		}
	}

	bare := ErrorPanel(theme, "boom", "", "")
	if len(bare) != 3 {
		t.Errorf("ErrorPanel with no hint/ref: got %d lines, want 3 (blank, headline, blank): %q", len(bare), bare)
	}
}

// ErrorScreen prints exactly what ErrorPanel builds — the rewrite from
// interleaved Fprintln calls to a single []string must not change
// output.
func TestErrorScreenMatchesErrorPanel(t *testing.T) {
	theme := GetTheme("minimal", false)
	err := errors.New("project name \"2bad\" must start with a letter")

	var buf strings.Builder
	ErrorScreen(&buf, theme, err)

	want := strings.Join(ErrorPanel(theme, errorMessage(err), suggestHint(err), docsRef(err)), "\n") + "\n"
	if buf.String() != want {
		t.Errorf("ErrorScreen output = %q, want %q", buf.String(), want)
	}
}

// successLines always leads with the "Generated <name>" line and
// includes the file tree and next steps when present.
func TestSuccessLinesShape(t *testing.T) {
	theme := GetTheme("minimal", false)
	s := engine.Summary{
		Template:            "go-rest-api",
		FilesWritten:        []string{"go.mod", "main.go"},
		CapabilitiesApplied: []string{"git-init"},
		NextSteps:           []string{"go build ./...", "go test ./..."},
	}

	lines := successLines(theme, "my-app", s, 0)
	if len(lines) == 0 {
		t.Fatal("successLines returned no lines")
	}
	if !strings.Contains(lines[0], "Generated my-app") {
		t.Errorf("successLines[0] = %q, want it to contain %q", lines[0], "Generated my-app")
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "go.mod") || !strings.Contains(joined, "main.go") {
		t.Errorf("successLines should list every written file:\n%s", joined)
	}
	if !strings.Contains(joined, "cd my-app") {
		t.Errorf("successLines should prepend a \"cd my-app\" next step:\n%s", joined)
	}
	if strings.Contains(joined, "completed in") {
		t.Errorf("successLines should omit the slow-run note under 5s:\n%s", joined)
	}

	slow := successLines(theme, "my-app", s, 6*time.Second)
	if !strings.Contains(strings.Join(slow, "\n"), "completed in") {
		t.Error("successLines should include a slow-run note over 5s")
	}
}
