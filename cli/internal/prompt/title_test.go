package prompt

import "testing"

// SetTitle/RestoreTitle must be no-ops when stdout is not a terminal:
// piped output stays deterministic and golden transcripts never contain
// escape sequences (design-system §6, §8.5). The test environment's
// stdout is not a terminal, so both calls must complete without
// panicking and without touching the pipe.
func TestSetTitleOffTTYNoop(t *testing.T) {
	SetTitle("Lumo — new project")
	RestoreTitle()
	SetTitle("Lumo — error")
	RestoreTitle()
}
