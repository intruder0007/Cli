package prompt

import (
	"errors"
	"fmt"
	"io"
)

// ErrCancelled is returned by SelectMenu/MultiSelectMenu when the user
// cancels (Ctrl+C, Esc, or 'q') instead of confirming a selection.
var ErrCancelled = errors.New("cancelled")

type key int

const (
	keyUp key = iota
	keyDown
	keyEnter
	keySpace
	keyCancel
	keyOther
)

// readByte reads exactly one byte from r.
func readByte(r io.Reader) (byte, error) {
	buf := make([]byte, 1)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return buf[0], nil
}

// readKey reads one logical keypress from r, recognizing ANSI arrow-key
// escape sequences (ESC [ A/B for up/down) alongside plain bytes. A bare
// ESC (not followed by '[') is treated as cancel, same as Ctrl+C or 'q'.
// j/k are accepted as vim-style down/up, a common terminal convention.
func readKey(r io.Reader) (key, error) {
	b, err := readByte(r)
	if err != nil {
		return keyOther, err
	}
	switch b {
	case 3: // Ctrl+C — raw mode disables the terminal's own SIGINT
		// generation, so this arrives as a plain byte, not a signal.
		return keyCancel, nil
	case 27: // ESC
		b2, err := readByte(r)
		if err != nil || b2 != '[' {
			return keyCancel, nil
		}
		b3, err := readByte(r)
		if err != nil {
			return keyOther, err
		}
		switch b3 {
		case 'A':
			return keyUp, nil
		case 'B':
			return keyDown, nil
		}
		return keyOther, nil
	case '\r', '\n':
		return keyEnter, nil
	case ' ':
		return keySpace, nil
	case 'k':
		return keyUp, nil
	case 'j':
		return keyDown, nil
	case 'q':
		return keyCancel, nil
	default:
		return keyOther, nil
	}
}

func moveCursorUp(w io.Writer, n int) {
	if n > 0 {
		fmt.Fprintf(w, "\x1b[%dA", n)
	}
}

func clearLine(w io.Writer) {
	fmt.Fprint(w, "\r\x1b[2K")
}

// renderLines writes label + one line per option, redrawing in place
// (moving the cursor back up and clearing) on every call after the
// first. Returns the line count, for the next call's redraw.
func renderLines(w io.Writer, t Theme, label string, lines []string, first bool) int {
	if !first {
		moveCursorUp(w, len(lines)+1)
	}
	clearLine(w)
	fmt.Fprintln(w, t.Header(label+":"))
	for _, l := range lines {
		clearLine(w)
		fmt.Fprintln(w, l)
	}
	return len(lines) + 1
}

func optionLine(t Theme, o option, highlighted bool, box string) string {
	prefix := "  "
	if highlighted {
		prefix = t.Cursor + " "
	}
	name := o.Name
	if !o.Available {
		name = t.Dim(name + " (coming soon)")
	}
	if box != "" {
		return prefix + box + " " + name
	}
	return prefix + name
}

// firstAvailable returns the index of the first available option
// matching defaultID, or the first available option at all, or 0.
func firstAvailable(opts []option, defaultID string) int {
	for i, o := range opts {
		if o.ID == defaultID && o.Available {
			return i
		}
	}
	for i, o := range opts {
		if o.Available {
			return i
		}
	}
	return 0
}

// SelectMenu renders opts and lets the user pick one available option
// with arrow keys (or j/k) + enter, redrawing in place. r must yield raw
// key bytes (a terminal in raw mode, or — in tests — a bytes.Reader with
// a simulated key sequence).
func SelectMenu(w io.Writer, r io.Reader, t Theme, label string, opts []option, defaultID string) (string, error) {
	cursor := firstAvailable(opts, defaultID)
	first := true

	redraw := func() {
		lines := make([]string, len(opts))
		for i, o := range opts {
			lines[i] = optionLine(t, o, i == cursor, "")
		}
		renderLines(w, t, label, lines, first)
		first = false
	}
	redraw()

	for {
		k, err := readKey(r)
		if err != nil {
			return "", err
		}
		switch k {
		case keyCancel:
			return "", ErrCancelled
		case keyUp:
			cursor = prevAvailable(opts, cursor)
			redraw()
		case keyDown:
			cursor = nextAvailable(opts, cursor)
			redraw()
		case keyEnter:
			if opts[cursor].Available {
				return opts[cursor].ID, nil
			}
		}
	}
}

func nextAvailable(opts []option, from int) int {
	for i := 1; i <= len(opts); i++ {
		idx := (from + i) % len(opts)
		if opts[idx].Available {
			return idx
		}
	}
	return from
}

func prevAvailable(opts []option, from int) int {
	for i := 1; i <= len(opts); i++ {
		idx := (from - i + len(opts)) % len(opts)
		if opts[idx].Available {
			return idx
		}
	}
	return from
}

// MultiSelectMenu renders opts with checkboxes; arrow keys (or j/k) move
// the cursor, space toggles the highlighted option, enter confirms. All
// options are expected to be available (V1 has no "coming soon"
// capabilities) but unavailable ones are shown dimmed and un-toggleable,
// for forward-compatibility with roadmap.md's future capability list.
func MultiSelectMenu(w io.Writer, r io.Reader, t Theme, label string, opts []option) ([]string, error) {
	cursor := firstAvailable(opts, "")
	selected := make(map[int]bool, len(opts))
	first := true

	redraw := func() {
		lines := make([]string, len(opts))
		for i, o := range opts {
			box := t.Unchecked
			if selected[i] {
				box = t.Checked
			}
			lines[i] = optionLine(t, o, i == cursor, box)
		}
		renderLines(w, t, label, lines, first)
		first = false
	}
	redraw()

	for {
		k, err := readKey(r)
		if err != nil {
			return nil, err
		}
		switch k {
		case keyCancel:
			return nil, ErrCancelled
		case keyUp:
			cursor = prevAvailable(opts, cursor)
			redraw()
		case keyDown:
			cursor = nextAvailable(opts, cursor)
			redraw()
		case keySpace:
			if opts[cursor].Available {
				selected[cursor] = !selected[cursor]
				redraw()
			}
		case keyEnter:
			var out []string
			for i, o := range opts {
				if selected[i] {
					out = append(out, o.ID)
				}
			}
			return out, nil
		}
	}
}
