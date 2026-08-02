package prompt

import (
	"errors"
	"fmt"
	"io"
	"strings"
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

// pushbackReader wraps an io.Reader with a one-byte unread buffer. The
// menus use it so that a bare ESC doesn't swallow the byte after it:
// readKeyByte must look one byte ahead to recognize arrow sequences,
// and when that lookahead turns out to be a plain key (ESC followed by
// Enter, say), it must be handed back to the next read.
type pushbackReader struct {
	r      io.Reader
	peeked []byte
}

func (p *pushbackReader) Read(b []byte) (int, error) {
	if len(p.peeked) > 0 {
		b[0] = p.peeked[0]
		p.peeked = p.peeked[1:]
		return 1, nil
	}
	return p.r.Read(b)
}

func (p *pushbackReader) unread(b byte) {
	p.peeked = append(p.peeked, b)
}

// readKeyByte reads one logical keypress from r, like readKey, but also
// returns the raw byte that produced it. The menus need the byte to
// distinguish filter characters (any printable) from backspace, and a
// bare ESC (clear the filter) from Ctrl+C/'q' (cancel). A bare ESC
// pushes any non-escape byte it consumed as lookahead back into r, so
// "ESC then Enter" doesn't lose the Enter.
func readKeyByte(r io.Reader) (key, byte, error) {
	b, err := readByte(r)
	if err != nil {
		return keyOther, 0, err
	}
	switch b {
	case 3: // Ctrl+C — raw mode disables the terminal's own SIGINT
		// generation, so this arrives as a plain byte, not a signal.
		return keyCancel, b, nil
	case 27: // ESC
		b2, err := readByte(r)
		if err != nil || b2 != '[' {
			if err == nil {
				if p, ok := r.(*pushbackReader); ok {
					p.unread(b2)
				}
			}
			return keyCancel, b, nil
		}
		b3, err := readByte(r)
		if err != nil {
			return keyOther, b, err
		}
		switch b3 {
		case 'A':
			return keyUp, b, nil
		case 'B':
			return keyDown, b, nil
		}
		return keyOther, b, nil
	case '\r', '\n':
		return keyEnter, b, nil
	case ' ':
		return keySpace, b, nil
	case 'k':
		return keyUp, b, nil
	case 'j':
		return keyDown, b, nil
	case 'q':
		return keyCancel, b, nil
	default:
		return keyOther, b, nil
	}
}

// readKey reads one logical keypress from r, recognizing ANSI arrow-key
// escape sequences (ESC [ A/B for up/down) alongside plain bytes. A bare
// ESC (not followed by '[') is treated as cancel, same as Ctrl+C or 'q'.
// j/k are accepted as vim-style down/up, a common terminal convention.
func readKey(r io.Reader) (key, error) {
	k, _, err := readKeyByte(r)
	return k, err
}

func moveCursorUp(w io.Writer, n int) {
	if n > 0 {
		fmt.Fprintf(w, "\x1b[%dA", n)
	}
}

func clearLine(w io.Writer) {
	fmt.Fprint(w, "\r\x1b[2K")
}

// renderLines writes label + one line per option + an optional footer
// (the key-binding hint), redrawing in place (moving the cursor back up
// and clearing) on every call after the first. Returns the line count,
// for the next call's redraw.
func renderLines(w io.Writer, t Theme, label string, lines []string, footer string, first bool) int {
	if !first {
		moveCursorUp(w, len(lines)+2)
	}
	clearLine(w)
	fmt.Fprintln(w, t.Header(label+":"))
	for _, l := range lines {
		clearLine(w)
		fmt.Fprintln(w, l)
	}
	if footer != "" {
		clearLine(w)
		fmt.Fprintln(w, footer)
	}
	return len(lines) + 2
}

// menuHint is the one-line key guide under every menu. It mentions the
// filter behavior whenever a filter is active, since that's when the
// bindings change meaning.
func menuHint(t Theme, filter string) string {
	nav := "up/down/j/k navigate"
	if t.UseIcons {
		nav = "↑↓/j/k navigate"
	}
	hint := nav + " · enter select · type to filter · esc cancel"
	if filter != "" {
		hint += " · backspace clear"
	}
	return t.Dim(hint)
}

// fuzzyMatches reports whether needle is a case-insensitive subsequence
// of hay — "rbapi" matches "REST API (node:http)" — the match rule for
// menu type-ahead filtering. An empty needle matches everything.
func fuzzyMatches(hay, needle string) bool {
	h := strings.ToLower(hay)
	for _, r := range strings.ToLower(needle) {
		idx := strings.IndexRune(h, r)
		if idx < 0 {
			return false
		}
		h = h[idx+1:]
	}
	return true
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

// firstVisiblePos returns the position (index into visible) of the
// first available option matching defaultID, or the first available
// option at all, or 0.
func firstVisiblePos(opts []option, visible []int, defaultID string) int {
	for pos, idx := range visible {
		if opts[idx].ID == defaultID && opts[idx].Available {
			return pos
		}
	}
	for pos, idx := range visible {
		if opts[idx].Available {
			return pos
		}
	}
	return 0
}

// nextVisiblePos/prevVisiblePos move one position through visible,
// skipping unavailable options, wrapping around.
func nextVisiblePos(opts []option, visible []int, from int) int {
	for i := 1; i <= len(visible); i++ {
		pos := (from + i) % len(visible)
		if opts[visible[pos]].Available {
			return pos
		}
	}
	return from
}

func prevVisiblePos(opts []option, visible []int, from int) int {
	for i := 1; i <= len(visible); i++ {
		pos := (from - i + len(visible)) % len(visible)
		if opts[visible[pos]].Available {
			return pos
		}
	}
	return from
}

// applyFilter rebuilds visible from opts for the current filter and
// resets the cursor to the first match. Fuzzy search lets the user type
// any letters to narrow the list (e.g. "rbapi" → "REST API") instead of
// arrow-keying through everything.
func applyFilter(opts []option, visible []int, filter string) ([]int, int) {
	visible = visible[:0]
	for i, o := range opts {
		if fuzzyMatches(o.Name, filter) {
			visible = append(visible, i)
		}
	}
	return visible, firstVisiblePos(opts, visible, "")
}

// SelectMenu renders opts and lets the user pick one available option
// with arrow keys (or j/k) + enter, redrawing in place. Typing narrows
// the list by fuzzy subsequence match; backspace edits the filter; a
// bare ESC clears the filter first and cancels only when it's already
// empty; Ctrl+C or 'q' always cancels. r must yield raw key bytes (a
// terminal in raw mode, or — in tests — a bytes.Reader with a simulated
// key sequence).
func SelectMenu(w io.Writer, r io.Reader, t Theme, label string, opts []option, defaultID string) (string, error) {
	visible := make([]int, len(opts))
	for i := range opts {
		visible[i] = i
	}
	cursor := firstVisiblePos(opts, visible, defaultID)
	filter := ""
	first := true

	redraw := func() {
		lines := make([]string, 0, len(visible))
		for pos, idx := range visible {
			lines = append(lines, optionLine(t, opts[idx], pos == cursor, ""))
		}
		if len(visible) == 0 && filter != "" {
			lines = append(lines, t.Dim(fmt.Sprintf("no matches for %q", filter)))
		}
		renderLines(w, t, label, lines, menuHint(t, filter), first)
		first = false
	}
	redraw()
	r = &pushbackReader{r: r}
	for {
		k, b, err := readKeyByte(r)
		if err != nil {
			return "", err
		}
		switch k {
		case keyCancel:
			if b == 27 && filter != "" {
				// Bare ESC first clears the filter — the menu itself
				// cancels only on a second ESC (or Ctrl+C/'q').
				filter = ""
				visible, cursor = applyFilter(opts, visible, filter)
				redraw()
				continue
			}
			return "", ErrCancelled
		case keyUp:
			if len(visible) > 0 {
				cursor = prevVisiblePos(opts, visible, cursor)
				redraw()
			}
		case keyDown:
			if len(visible) > 0 {
				cursor = nextVisiblePos(opts, visible, cursor)
				redraw()
			}
		case keyEnter:
			if len(visible) > 0 && opts[visible[cursor]].Available {
				return opts[visible[cursor]].ID, nil
			}
		case keyOther:
			switch {
			case b == 127 || b == 8: // backspace/delete
				if filter != "" {
					filter = filter[:len(filter)-1]
					visible, cursor = applyFilter(opts, visible, filter)
					redraw()
				}
			case b >= 32 && b <= 126:
				filter += string(b)
				visible, cursor = applyFilter(opts, visible, filter)
				redraw()
			}
		}
	}
}

// MultiSelectMenu renders opts with checkboxes; arrow keys (or j/k) move
// the cursor, space toggles the highlighted option, enter confirms.
// Typing narrows the list by fuzzy subsequence match, exactly like
// SelectMenu. All options are expected to be available (V1 has no
// "coming soon" capabilities) but unavailable ones are shown dimmed and
// un-toggleable, for forward-compatibility with roadmap.md's future
// capability list.
func MultiSelectMenu(w io.Writer, r io.Reader, t Theme, label string, opts []option) ([]string, error) {
	visible := make([]int, len(opts))
	for i := range opts {
		visible[i] = i
	}
	cursor := firstVisiblePos(opts, visible, "")
	selected := make(map[int]bool, len(opts))
	filter := ""
	first := true

	redraw := func() {
		lines := make([]string, 0, len(visible))
		for pos, idx := range visible {
			box := t.Unchecked
			if selected[idx] {
				box = t.Checked
			}
			lines = append(lines, optionLine(t, opts[idx], pos == cursor, box))
		}
		if len(visible) == 0 && filter != "" {
			lines = append(lines, t.Dim(fmt.Sprintf("no matches for %q", filter)))
		}
		renderLines(w, t, label, lines, menuHint(t, filter), first)
		first = false
	}
	redraw()
	r = &pushbackReader{r: r}
	for {
		k, b, err := readKeyByte(r)
		if err != nil {
			return nil, err
		}
		switch k {
		case keyCancel:
			if b == 27 && filter != "" {
				filter = ""
				visible, cursor = applyFilter(opts, visible, filter)
				redraw()
				continue
			}
			return nil, ErrCancelled
		case keyUp:
			if len(visible) > 0 {
				cursor = prevVisiblePos(opts, visible, cursor)
				redraw()
			}
		case keyDown:
			if len(visible) > 0 {
				cursor = nextVisiblePos(opts, visible, cursor)
				redraw()
			}
		case keySpace:
			if len(visible) > 0 && opts[visible[cursor]].Available {
				selected[visible[cursor]] = !selected[visible[cursor]]
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
		case keyOther:
			switch {
			case b == 127 || b == 8: // backspace/delete
				if filter != "" {
					filter = filter[:len(filter)-1]
					visible, cursor = applyFilter(opts, visible, filter)
					redraw()
				}
			case b >= 32 && b <= 126:
				filter += string(b)
				visible, cursor = applyFilter(opts, visible, filter)
				redraw()
			}
		}
	}
}
