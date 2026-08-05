package prompt

import "testing"

func TestStepChrome(t *testing.T) {
	got := StepChrome(2, 5, "Theme")
	want := "Step 2 of 5 · Theme"
	if got != want {
		t.Errorf("StepChrome(2, 5, %q) = %q, want %q", "Theme", got, want)
	}
}

func TestStepChromeEmptyName(t *testing.T) {
	got := StepChrome(1, 3, "")
	want := "Step 1 of 3 · Step"
	if got != want {
		t.Errorf("StepChrome with empty name = %q, want %q", got, want)
	}
}

func TestWizardStepNameMatchesScreenSpec(t *testing.T) {
	// screen-spec.md-derived titles, by step constant (location is new,
	// not yet in the S02-S07 doc numbering).
	cases := map[int]string{
		stepName:     "Project name",
		stepLocation: "Location",
		stepTheme:    "Theme",
		stepType:     "Project type",
		stepLang:     "Language",
		stepFw:       "Framework",
		7:            "Capabilities",
		8:            "Confirm",
		99:           "Step",
	}
	for st, want := range cases {
		if got := wizardStepName(st); got != want {
			t.Errorf("wizardStepName(%d) = %q, want %q", st, got, want)
		}
	}
}
