package prompt

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intruder0007/Lumo/core/config"
)

// The wizard's step arithmetic drives the "Step N of M · name" titles
// and which screen each step renders — pinned here so the confirm step
// lands last whether or not capability plugins are installed.
func TestWizardSteps(t *testing.T) {
	withCaps := WizardSpec{Templates: []TemplateSpec{{}}, Capabilities: []CapabilitySpec{{ID: "git-init"}}}
	if got := wizardSteps(withCaps); got != 8 {
		t.Errorf("wizardSteps with capabilities = %d, want 8", got)
	}
	noCaps := WizardSpec{Templates: []TemplateSpec{{}}}
	if got := wizardSteps(noCaps); got != 7 {
		t.Errorf("wizardSteps without capabilities = %d, want 7", got)
	}
}

func TestWizardStepTitle(t *testing.T) {
	for st, want := range map[int]string{
		1: "Step 1 of 8 · Project name",
		2: "Step 2 of 8 · Location",
		3: "Step 3 of 8 · Theme",
		4: "Step 4 of 8 · Project type",
		5: "Step 5 of 8 · Language",
		6: "Step 6 of 8 · Framework",
		7: "Step 7 of 8 · Capabilities",
		8: "Step 8 of 8 · Confirm",
	} {
		if got := wizardStepTitle(st, 8); got != want {
			t.Errorf("wizardStepTitle(%d) = %q, want %q", st, got, want)
		}
	}
}

// The location step's fallback default must never be the directory the
// binary happens to be running from — that's the exact bug it exists
// to fix (a bare name resolving via os.Getwd(), which on Windows is the
// double-clicked .exe's own folder). defaultProjectsDir is home-based
// and doesn't touch os.Executable/os.Getwd at all.
func TestDefaultProjectsDirNeverExecutableDir(t *testing.T) {
	exeDir := "."
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}

	got := defaultProjectsDir()
	if got == exeDir {
		t.Errorf("defaultProjectsDir() = %q, must never equal the executable's own directory (%q)", got, exeDir)
	}
	if got == "" || got == "." {
		home, err := os.UserHomeDir()
		if err == nil {
			t.Errorf("defaultProjectsDir() = %q, want a home-relative path (home=%q) when UserHomeDir succeeds", got, home)
		}
	}
}

// combinedTarget is what threads the wizard's separately-collected
// location and name into the single path config.Answers.ProjectName
// carries downstream (see cli/main.go's resolveTargetPath).
func TestCombinedTarget(t *testing.T) {
	if got, want := combinedTarget(`C:\Users\me\Projects`, "my-app"), filepath.Join(`C:\Users\me\Projects`, "my-app"); got != want {
		t.Errorf("combinedTarget = %q, want %q", got, want)
	}
	if got := combinedTarget("", "my-app"); got != "my-app" {
		t.Errorf("combinedTarget with empty location = %q, want bare name %q", got, "my-app")
	}
}

// A location entered on one wizard run must be offered as the pre-fill
// on the next — persistProjectsDir/LoadConfig is the same round-trip
// the wizard's stepLocation relies on.
func TestPersistProjectsDirRoundTrip(t *testing.T) {
	withTempConfigDir(t)

	cfg, _ := LoadConfig()
	if cfg.DefaultProjectsDir != "" {
		t.Fatalf("fresh config: got DefaultProjectsDir=%q, want empty", cfg.DefaultProjectsDir)
	}

	persistProjectsDir(`C:\Users\me\Projects`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig after persistProjectsDir: %v", err)
	}
	if want := `C:\Users\me\Projects`; cfg.DefaultProjectsDir != want {
		t.Errorf("second run's pre-fill: got %q, want %q (the first run's persisted choice)", cfg.DefaultProjectsDir, want)
	}

	// A second, different choice overwrites the first — "last-used",
	// not "first-ever".
	persistProjectsDir(`D:\code`)
	cfg, _ = LoadConfig()
	if want := `D:\code`; cfg.DefaultProjectsDir != want {
		t.Errorf("after a second persistProjectsDir: got %q, want %q", cfg.DefaultProjectsDir, want)
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
		err := confirmStep(o, strings.NewReader(tc.input), GetTheme("minimal", false), 8, 8, config.Answers{})
		if err != tc.want {
			t.Errorf("confirmStep(%s): got %v, want %v", tc.name, err, tc.want)
		}
	}
}
