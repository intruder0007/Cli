package prompt

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intruder0007/Lumo/core/config"
)

// The wizard's step arithmetic drives the "Step N of M · name" titles
// and which screen each step renders — pinned here so the confirm step
// lands last whether or not capability plugins are installed.
func TestWizardSteps(t *testing.T) {
	withCaps := WizardSpec{Templates: []TemplateSpec{{}}, Capabilities: []CapabilitySpec{{ID: "git-init"}}}
	if got := wizardSteps(withCaps); got != 7 {
		t.Errorf("wizardSteps with capabilities = %d, want 7", got)
	}
	noCaps := WizardSpec{Templates: []TemplateSpec{{}}}
	if got := wizardSteps(noCaps); got != 6 {
		t.Errorf("wizardSteps without capabilities = %d, want 6", got)
	}
}

func TestWizardStepTitle(t *testing.T) {
	for st, want := range map[int]string{
		1: "Step 1 of 7 · Project name",
		2: "Step 2 of 7 · Theme",
		3: "Step 3 of 7 · Project type",
		4: "Step 4 of 7 · Language",
		5: "Step 5 of 7 · Framework",
		6: "Step 6 of 7 · Capabilities",
		7: "Step 7 of 7 · Confirm",
	} {
		if got := wizardStepTitle(st, 7); got != want {
			t.Errorf("wizardStepTitle(%d) = %q, want %q", st, got, want)
		}
	}
}

// confirmStep maps the confirm screen's keys onto wizard navigation:
// Enter confirms (errConfirmed), a back key (h / ←) walks back a step
// (ErrBack), Ctrl+C/q cancels (ErrCancelled) — so the outer loop never
// advances past the last step.
func TestConfirmStep(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  error
	}{
		{"enter", "\r", errConfirmed},
		{"h", "h", ErrBack},
		{"q", "q", ErrCancelled},
		{"ctrl-c", "\x03", ErrCancelled},
	} {
		var out bytes.Buffer
		o := NewOutput(&out)
		err := confirmStep(o, strings.NewReader(tc.input), GetTheme("minimal", false), 7, 7, config.Answers{})
		if err != tc.want {
			t.Errorf("confirmStep(%s): got %v, want %v", tc.name, err, tc.want)
		}
	}
}
