package prompt

import (
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

// Output is the rendering engine's one write path. It wraps the
// underlying stream, knows whether it is a real terminal, and owns
// every cursor/clear escape sequence — so piped and CI output never
// contains them. All writes are mutex-serialized, which is what lets
// the progress engine's animation goroutine and the input loop share
// one stream without interleaving.
type Output struct {
	w   io.Writer
	tty bool
	mu  sync.Mutex
}

// NewOutput wraps w, detecting a real terminal when w is an *os.File.
func NewOutput(w io.Writer) *Output {
	tty := false
	if f, ok := w.(*os.File); ok {
		tty = term.IsTerminal(int(f.Fd()))
	}
	return &Output{w: w, tty: tty}
}

// TTY reports whether the underlying stream is a real terminal.
func (o *Output) TTY() bool { return o.tty }

// Write implements io.Writer (so an Output can be passed anywhere a
// writer is expected), routing through the same serialized write path.
func (o *Output) Write(p []byte) (int, error) {
	o.writeStr(string(p))
	return len(p), nil
}

// writeStr writes s verbatim, serialized against concurrent writers.
func (o *Output) writeStr(s string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	fmt.Fprint(o.w, s)
}

// Println writes s plus a newline.
func (o *Output) Println(s string) { o.writeStr(s + "\n") }

// Width returns the terminal width, or 0 when not a terminal (the
// layout engine then clamps to its own 80-column default).
func (o *Output) Width() int {
	if !o.tty {
		return 0
	}
	if _, cols, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		return cols
	}
	return 0
}

// The escape sequences below are all TTY-gated: off a terminal they are
// no-ops, keeping piped transcripts byte-stable and escape-free.

// ClearScreen moves the cursor home and clears the whole screen.
// Screens use it to transition atomically (design-spec §3.1):
// clear + redraw, never append.
func (o *Output) ClearScreen() {
	if !o.tty {
		return
	}
	o.writeStr("\x1b[2J\x1b[H")
}

// HideCursor and ShowCursor suppress/restore the terminal cursor while
// a widget manages its own caret.
func (o *Output) HideCursor() {
	if o.tty {
		o.writeStr("\x1b[?25l")
	}
}

func (o *Output) ShowCursor() {
	if o.tty {
		o.writeStr("\x1b[?25h")
	}
}

// MoveUp moves the cursor up n rows.
func (o *Output) MoveUp(n int) {
	if o.tty && n > 0 {
		o.writeStr(fmt.Sprintf("\x1b[%dA", n))
	}
}

// ClearLine returns the cursor to column 0 and clears the line.
func (o *Output) ClearLine() {
	if o.tty {
		o.writeStr("\r\x1b[2K")
	}
}

// ClearBelow clears from the cursor to the end of the screen.
func (o *Output) ClearBelow() {
	if o.tty {
		o.writeStr("\x1b[J")
	}
}

// CarriageReturn returns to column 0 without clearing — the off-terminal
// overtype primitive (a bare \r, never an escape code).
func (o *Output) CarriageReturn() {
	o.writeStr("\r")
}

// Region is the in-place redraw primitive. It renders a block of lines
// and, on later draws, moves back to the block's top and clears below
// before rewriting — so widgets (menus, the progress dashboard) update
// atomically without flicker and without touching lines above the
// region. Off a terminal it simply appends (deterministic transcripts).
type Region struct {
	out  *Output
	rows int
	used bool
}

// NewRegion returns a Region drawing in place on o.
func NewRegion(o *Output) *Region { return &Region{out: o} }

// Draw renders lines, replacing the previous frame in place. Each line
// is written with a trailing newline (blank lines included).
func (r *Region) Draw(lines []string) {
	if !r.out.tty {
		for _, l := range lines {
			r.out.Println(l)
		}
		return
	}
	if r.used {
		r.out.MoveUp(r.rows)
		r.out.ClearBelow()
	}
	r.rows = len(lines)
	r.used = true
	for _, l := range lines {
		r.out.Println(l)
	}
}
