package prompt

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestReadKeyArrowsAndControls(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
		want  key
	}{
		{"up arrow", []byte("\x1b[A"), keyUp},
		{"down arrow", []byte("\x1b[B"), keyDown},
		{"vim k (up)", []byte("k"), keyUp},
		{"vim j (down)", []byte("j"), keyDown},
		{"enter (CR)", []byte("\r"), keyEnter},
		{"enter (LF)", []byte("\n"), keyEnter},
		{"space", []byte(" "), keySpace},
		{"ctrl-c", []byte{3}, keyCancel},
		{"bare esc", []byte{27}, keyCancel},
		{"q", []byte("q"), keyCancel},
		{"unrecognized letter", []byte("x"), keyOther},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := readKey(bytes.NewReader(c.input))
			if err != nil {
				t.Fatalf("readKey(%q): unexpected error: %v", c.input, err)
			}
			if got != c.want {
				t.Errorf("readKey(%q) = %v, want %v", c.input, got, c.want)
			}
		})
	}
}

func TestReadKeyEOF(t *testing.T) {
	_, err := readKey(bytes.NewReader(nil))
	if !errors.Is(err, io.EOF) {
		t.Errorf("readKey on empty reader: got err=%v, want io.EOF", err)
	}
}

var testOpts = []option{
	{"a", "Alpha", true},
	{"b", "Beta", true},
	{"c", "Gamma (coming soon)", false},
}

func TestSelectMenuArrowNavigation(t *testing.T) {
	// down, down, enter — from default "a": down -> b, down -> wraps
	// past unavailable "c" back to "a", enter confirms "a"... but since
	// down skips unavailable options, two downs from "a" should land
	// back on "a" (a -> b -> a, skipping c). Confirms the "skip
	// unavailable options" behavior, not just raw index math.
	in := bytes.NewReader([]byte("\x1b[B\x1b[B\r"))
	var out bytes.Buffer
	got, err := SelectMenu(&out, in, GetTheme("minimal", false), "Pick", testOpts, "a")
	if err != nil {
		t.Fatalf("SelectMenu: %v", err)
	}
	if got != "a" {
		t.Errorf("two downs from \"a\" (skipping unavailable \"c\") then enter: got %q, want %q", got, "a")
	}
}

func TestSelectMenuSkipsUnavailableOnEnter(t *testing.T) {
	// enter immediately: should return the default, not silently accept
	// an unavailable option even if it were somehow the cursor position.
	in := bytes.NewReader([]byte("\r"))
	var out bytes.Buffer
	got, err := SelectMenu(&out, in, GetTheme("minimal", false), "Pick", testOpts, "b")
	if err != nil {
		t.Fatalf("SelectMenu: %v", err)
	}
	if got != "b" {
		t.Errorf("immediate enter with default \"b\": got %q, want %q", got, "b")
	}
}

func TestSelectMenuCancel(t *testing.T) {
	in := bytes.NewReader([]byte{3}) // Ctrl+C
	var out bytes.Buffer
	_, err := SelectMenu(&out, in, GetTheme("minimal", false), "Pick", testOpts, "a")
	if err != ErrCancelled {
		t.Errorf("SelectMenu with Ctrl+C: got err=%v, want ErrCancelled", err)
	}
}

func TestSelectMenuFuzzyFilter(t *testing.T) {
	// Typing "et" — a fuzzy subsequence of "Beta" — filters the list to
	// Beta; enter confirms it.
	in := bytes.NewReader([]byte("et\r"))
	var out bytes.Buffer
	got, err := SelectMenu(&out, in, GetTheme("minimal", false), "Pick", testOpts, "a")
	if err != nil {
		t.Fatalf("SelectMenu: %v", err)
	}
	if got != "b" {
		t.Errorf("typed \"et\" then enter: got %q, want %q (fuzzy match on \"Beta\")", got, "b")
	}
}

func TestSelectMenuEscClearsFilterThenCancels(t *testing.T) {
	// "et" filters to Beta; a bare ESC clears the filter instead of
	// cancelling, so the next enter confirms the default "a".
	in := bytes.NewReader([]byte("et\x1b\r"))
	var out bytes.Buffer
	got, err := SelectMenu(&out, in, GetTheme("minimal", false), "Pick", testOpts, "a")
	if err != nil {
		t.Fatalf("SelectMenu: %v", err)
	}
	if got != "a" {
		t.Errorf("typed \"et\", ESC, enter: got %q, want %q (ESC clears the filter, doesn't cancel)", got, "a")
	}
}

func TestSelectMenuFilterNoMatchesEnterDoesNothing(t *testing.T) {
	// "zzz" matches nothing and enter must not select (or panic);
	// backspaces clear the filter, then enter picks the default.
	in := bytes.NewReader([]byte("zzz\x7f\x7f\x7f\r"))
	var out bytes.Buffer
	got, err := SelectMenu(&out, in, GetTheme("minimal", false), "Pick", testOpts, "a")
	if err != nil {
		t.Fatalf("SelectMenu: %v", err)
	}
	if got != "a" {
		t.Errorf("typed \"zzz\", cleared with backspace, enter: got %q, want %q", got, "a")
	}
}

func TestMultiSelectMenuFuzzyToggle(t *testing.T) {
	// Typing "al" filters to Alpha; space toggles it; enter confirms.
	in := bytes.NewReader([]byte("al \r"))
	var out bytes.Buffer
	got, err := MultiSelectMenu(&out, in, GetTheme("minimal", false), "Pick", testOpts)
	if err != nil {
		t.Fatalf("MultiSelectMenu: %v", err)
	}
	want := []string{"a"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("typed \"al\", space, enter: got %v, want %v", got, want)
	}
}

func TestMultiSelectMenuToggleAndConfirm(t *testing.T) {
	// space (select "a"), down, space (select "b"), enter.
	in := bytes.NewReader([]byte(" \x1b[B \r"))
	var out bytes.Buffer
	got, err := MultiSelectMenu(&out, in, GetTheme("minimal", false), "Pick", testOpts)
	if err != nil {
		t.Fatalf("MultiSelectMenu: %v", err)
	}
	want := []string{"a", "b"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("MultiSelectMenu space,down,space,enter: got %v, want %v", got, want)
	}
}

func TestMultiSelectMenuToggleOff(t *testing.T) {
	// space (select "a"), space (deselect "a"), enter -> nothing selected.
	in := bytes.NewReader([]byte("  \r"))
	var out bytes.Buffer
	got, err := MultiSelectMenu(&out, in, GetTheme("minimal", false), "Pick", testOpts)
	if err != nil {
		t.Fatalf("MultiSelectMenu: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("toggle on then off then enter: got %v, want empty", got)
	}
}
