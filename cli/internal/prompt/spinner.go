package prompt

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// Spinner renders phase progress for a `lumo new` run. On a real
// terminal it animates a frame + the current phase label in place; when
// the output stream isn't a terminal (pipes, CI, tests) it degrades to
// one static "arrow label" line per phase, so scripts see the same
// phase structure with no escape codes. NO_COLOR/minimal themes use
// ASCII frames and no color, per the theme's UseIcons/UseColor.
type Spinner struct {
	w   io.Writer
	t   Theme
	tty bool

	mu       sync.Mutex
	label    string
	done     chan struct{}
	animDone chan struct{}
}

// NewSpinner returns a Spinner writing to w, animating only if w is a
// terminal.
func NewSpinner(w io.Writer, t Theme) *Spinner {
	f, ok := w.(*os.File)
	tty := ok && term.IsTerminal(int(f.Fd()))
	return &Spinner{w: w, t: t, tty: tty}
}

// Start begins animating (or, off a terminal, prints one static line)
// for the given phase. Calling Start while a previous phase is still
// animating finalizes that phase as successful first, so only one phase
// is ever visible.
func (s *Spinner) Start(label string) {
	s.finish(true)
	s.mu.Lock()
	s.label = label
	s.mu.Unlock()

	if !s.tty {
		fmt.Fprintln(s.w, "  "+s.t.Accent(s.arrow())+" "+label)
		return
	}
	s.done = make(chan struct{})
	s.animDone = make(chan struct{})
	go s.animate()
}

// Finish stops the spinner and, when successful, replaces it with a
// "✔ label" line. On failure it only clears the animation — the caller
// renders the error (ErrorScreen), so a spinner frame never interleaves
// with the error text. Safe to call any number of times.
func (s *Spinner) Finish(success bool) {
	if s.tty {
		s.finish(success)
	}
}

// finish finalizes the in-flight phase. It is a no-op when no phase is
// animating (including the static-line non-TTY mode, where Start's line
// already stands alone).
func (s *Spinner) finish(success bool) {
	s.mu.Lock()
	if s.done == nil {
		s.mu.Unlock()
		return
	}
	label := s.label
	done := s.done
	animDone := s.animDone
	s.done = nil
	s.mu.Unlock()

	close(done)
	<-animDone
	fmt.Fprint(s.w, "\r\x1b[2K")
	if success {
		fmt.Fprintln(s.w, s.t.Success(label))
	}
}

func (s *Spinner) arrow() string {
	if s.t.UseIcons {
		return "→"
	}
	return "-"
}

func (s *Spinner) animate() {
	frames := s.t.Spinner
	if len(frames) == 0 {
		frames = []string{"|", "/", "-", "\\"}
	}
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	i := 0
	fmt.Fprint(s.w, "\r"+frames[i]+" "+s.currentLabel()+"\x1b[K")
	for {
		select {
		case <-s.done:
			close(s.animDone)
			return
		case <-tick.C:
			i = (i + 1) % len(frames)
			fmt.Fprint(s.w, "\r"+frames[i]+" "+s.currentLabel()+"\x1b[K")
		}
	}
}

func (s *Spinner) currentLabel() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.label
}
