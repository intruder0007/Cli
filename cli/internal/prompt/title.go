package prompt

import (
	"os"

	"golang.org/x/term"
)

// Terminal-title support (design-system §6): every interactive
// organism sets the window title to Lumo (with the activity), and
// RestoreTitle puts the previous title back on exit. Best effort:
// writing the title never fails a command.

var titleSet bool

// SetTitle sets the terminal window title, best effort. It is a no-op
// unless stdout is a terminal, so piped output never contains escape
// sequences and golden transcripts stay stable.
func SetTitle(title string) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	titleSet = true
	setTerminalTitle(title)
}

// RestoreTitle restores the title that was in effect before the first
// SetTitle call, best effort. No-op when no title was set (or stdout is
// not a terminal).
func RestoreTitle() {
	if !titleSet {
		return
	}
	titleSet = false
	restoreTerminalTitle()
}
