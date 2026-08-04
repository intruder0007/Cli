package prompt

import (
	"errors"
	"fmt"
	"strings"
	"testing"

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
