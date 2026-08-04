package prompt

import "fmt"

// chrome.go holds the wizard's step chrome: the "Step N of M · <name>"
// line every screen leads with (design-system §3, screen-spec.md
// S02-S07). It's the piece of the wizard that answers "where am I" —
// pulled out of wizard.go so it's a named, reusable, independently
// testable component rather than a private per-file helper.

// StepChrome renders the step-progress title line for step st of a
// total steps wizard (e.g. "Step 2 of 5 · Theme"). name should be the
// human label for st (see wizardStepName); an empty name renders as
// the generic "Step".
func StepChrome(st, steps int, name string) string {
	if name == "" {
		name = "Step"
	}
	return fmt.Sprintf("Step %d of %d · %s", st, steps, name)
}

// wizardStepName returns the human label for a wizard step number
// (wizard.go's step constants), used by StepChrome to build the full
// "Step N of M · <name>" title.
func wizardStepName(st int) string {
	switch st {
	case stepName:
		return "Project name"
	case stepTheme:
		return "Theme"
	case stepType:
		return "Project type"
	case stepLang:
		return "Language"
	case stepFw:
		return "Framework"
	case 6:
		return "Capabilities"
	case 7:
		return "Confirm"
	default:
		return "Step"
	}
}
