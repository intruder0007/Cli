package prompt

import (
	"runtime"
	"testing"
)

// withTempConfigDir redirects os.UserConfigDir() to t.TempDir() for the
// duration of the test, so LoadConfig/SaveConfig never touch the real
// user's config file.
func withTempConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", dir)
	} else {
		t.Setenv("XDG_CONFIG_HOME", dir)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	withTempConfigDir(t)

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig on a missing file should not error, got: %v", err)
	}
	if got.Theme != "" {
		t.Errorf("LoadConfig with no saved file: got Theme=%q, want empty", got.Theme)
	}
	if got.DefaultProjectsDir != "" {
		t.Errorf("LoadConfig with no saved file: got DefaultProjectsDir=%q, want empty", got.DefaultProjectsDir)
	}

	if err := SaveConfig(Config{Theme: "minimal", DefaultProjectsDir: `C:\Users\me\Projects`}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	got, err = LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig after save: %v", err)
	}
	if got.Theme != "minimal" {
		t.Errorf("LoadConfig after SaveConfig(Theme: \"minimal\"): got %q, want %q", got.Theme, "minimal")
	}
	if want := `C:\Users\me\Projects`; got.DefaultProjectsDir != want {
		t.Errorf("LoadConfig after SaveConfig(DefaultProjectsDir): got %q, want %q", got.DefaultProjectsDir, want)
	}
}
