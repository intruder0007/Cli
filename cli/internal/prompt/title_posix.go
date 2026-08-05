//go:build !windows

package prompt

import (
	"fmt"
	"os"
)

// On POSIX the previous title is not queryable without extra syscalls;
// restoring to the empty string asks the terminal to fall back to its
// default (usually the shell's own prompt title), which is the best
// available approximation.
func setTerminalTitle(title string) {
	fmt.Fprintf(os.Stdout, "\x1b]2;%s\x07", title)
}

func restoreTerminalTitle() {
	fmt.Fprint(os.Stdout, "\x1b]2;\x07")
}
