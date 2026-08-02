package prompt

// Theme is a named set of rendering choices: whether to use color/icons,
// which glyphs menu.go's widgets use for the cursor and multi-select
// checkboxes, and the spinner frames spinner.go animates during a run.
// A Theme is data, not behavior baked into the CLI — see RegisterTheme.
// The semantic color helpers (Primary/Accent/Warn/Border) are the design
// tokens the whole CLI renders with; a theme only chooses whether to
// emit them at all (UseColor), so NO_COLOR/--no-color keeps working
// regardless of which theme is active.
type Theme struct {
	Name     string
	UseColor bool
	UseIcons bool

	// Cursor marks the highlighted row in a menu.
	Cursor string
	// Checked/Unchecked are the multi-select checkbox glyphs.
	Checked, Unchecked string
	// Spinner frames animate the in-progress phase of a run (spinner.go).
	// Empty means an ASCII fallback is used.
	Spinner []string
}

var (
	themeRegistry = map[string]Theme{}
	themeOrder    []string
)

// RegisterTheme adds (or replaces) a theme by name. This is the seam a
// future theme-plugin mechanism would call into — see ADR-0007 and
// docs/architecture/roadmap.md. It's an in-process extension point today,
// not a subprocess plugin: a terminal color scheme doesn't need the full
// plugin protocol's isolation guarantees.
func RegisterTheme(t Theme) {
	if _, exists := themeRegistry[t.Name]; !exists {
		themeOrder = append(themeOrder, t.Name)
	}
	themeRegistry[t.Name] = t
}

// ThemeNames returns registered theme names in registration order.
func ThemeNames() []string {
	out := make([]string, len(themeOrder))
	copy(out, themeOrder)
	return out
}

// GetTheme looks up a theme by name, falling back to "default" if the
// name is unknown or empty. noColor forces both UseColor and UseIcons
// off regardless of the theme's own defaults (glyphs like ❯/◉ are color
// cues too — a plain terminal shouldn't show them either; minimal is
// already UseColor:false/UseIcons:false, so this is a no-op for it).
func GetTheme(name string, noColor bool) Theme {
	t, ok := themeRegistry[name]
	if !ok {
		t = themeRegistry["default"]
	}
	if noColor {
		t.UseColor = false
		t.UseIcons = false
	}
	return t
}

func init() {
	RegisterTheme(Theme{
		Name: "default", UseColor: true, UseIcons: true,
		Cursor: "❯", Checked: "◉", Unchecked: "○",
		Spinner: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	})
	RegisterTheme(Theme{
		Name: "minimal", UseColor: false, UseIcons: false,
		Cursor: ">", Checked: "[x]", Unchecked: "[ ]",
		Spinner: []string{"|", "/", "-", "\\"},
	})
}

func (t Theme) color(code, s string) string {
	if !t.UseColor {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

// Primary formats text in the brand color — the Lumo wordmark, section
// headers, and the wizard's header lines.
func (t Theme) Primary(s string) string { return t.color("1;36", s) }

// Accent highlights interactive elements — progress arrows, next steps.
func (t Theme) Accent(s string) string { return t.color("38;5;75", s) }

// Warn flags a non-fatal condition that still deserves attention.
func (t Theme) Warn(s string) string { return t.color("38;5;214", s) }

// Border renders structural glyphs (tree branches, dividers) in a
// quieter gray than the surrounding text.
func (t Theme) Border(s string) string { return t.color("38;5;240", s) }

// Success formats a success line. Every state has a text label — color
// and icons are additive, never the only signal.
func (t Theme) Success(msg string) string {
	label := "[OK] "
	if t.UseIcons {
		label = "✔ "
	}
	return t.color("32", label) + msg
}

// Failure formats a failure line.
func (t Theme) Failure(msg string) string {
	label := "[FAIL] "
	if t.UseIcons {
		label = "✗ "
	}
	return t.color("31", label) + msg
}

// Info formats an informational line (e.g. a generated file).
func (t Theme) Info(msg string) string {
	return msg
}

// Header formats a section header.
func (t Theme) Header(msg string) string {
	return t.Primary(msg)
}

// Dim formats de-emphasized text (e.g. "(coming soon)" hints).
func (t Theme) Dim(msg string) string {
	return t.color("2", msg)
}
