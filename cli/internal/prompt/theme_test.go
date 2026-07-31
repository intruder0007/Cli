package prompt

import "testing"

func TestGetThemeKnownAndUnknown(t *testing.T) {
	def := GetTheme("default", false)
	if !def.UseColor || !def.UseIcons {
		t.Errorf("default theme: got UseColor=%v UseIcons=%v, want both true", def.UseColor, def.UseIcons)
	}

	min := GetTheme("minimal", false)
	if min.UseColor || min.UseIcons {
		t.Errorf("minimal theme: got UseColor=%v UseIcons=%v, want both false", min.UseColor, min.UseIcons)
	}

	unknown := GetTheme("nope-not-a-theme", false)
	if unknown.Name != "default" {
		t.Errorf("unknown theme name: got %q, want fallback to \"default\"", unknown.Name)
	}
}

func TestGetThemeNoColorOverride(t *testing.T) {
	forced := GetTheme("default", true)
	if forced.UseColor {
		t.Error("noColor=true should force UseColor=false even for the default theme")
	}

	// minimal already has UseColor=false; noColor shouldn't change that.
	min := GetTheme("minimal", true)
	if min.UseColor {
		t.Error("minimal theme should stay UseColor=false regardless of noColor")
	}
}

func TestRegisterThemeIsExtensible(t *testing.T) {
	RegisterTheme(Theme{Name: "test-only-theme", UseColor: true, Cursor: "*"})
	defer delete(themeRegistry, "test-only-theme")

	got := GetTheme("test-only-theme", false)
	if got.Cursor != "*" {
		t.Errorf("got Cursor=%q, want %q — RegisterTheme should make new themes immediately available via GetTheme", got.Cursor, "*")
	}

	found := false
	for _, n := range ThemeNames() {
		if n == "test-only-theme" {
			found = true
		}
	}
	if !found {
		t.Error("ThemeNames() should include a theme registered via RegisterTheme")
	}
}

func TestThemeColorHelpers(t *testing.T) {
	colorTheme := Theme{UseColor: true, UseIcons: true}
	if got := colorTheme.Success("x"); got == "x" || got[0] != '\033' {
		t.Errorf("Success with UseColor=true should wrap in an ANSI code, got %q", got)
	}

	plain := Theme{UseColor: false, UseIcons: false}
	if got := plain.Success("x"); got != "[OK] x" {
		t.Errorf("Success with UseColor=false UseIcons=false: got %q, want %q", got, "[OK] x")
	}
	if got := plain.Failure("x"); got != "[FAIL] x" {
		t.Errorf("Failure with UseColor=false UseIcons=false: got %q, want %q", got, "[FAIL] x")
	}
}
