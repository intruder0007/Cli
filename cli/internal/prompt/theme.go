package prompt

// Theme is a named set of rendering choices: whether to use color/icons,
// and which glyphs menu.go's widgets use for the cursor and multi-select
// checkboxes. A Theme is data, not behavior baked into the CLI — see
// RegisterTheme.
type Theme struct {
	Name     string
	UseColor bool
	UseIcons bool

	// Cursor marks the highlighted row in a menu.
	Cursor string
	// Checked/Unchecked are the multi-select checkbox glyphs.
	Checked, Unchecked string
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
// name is unknown or empty. noColor forces UseColor off regardless of
// the theme's own default (minimal is already UseColor:false, so this
// is a no-op for it).
func GetTheme(name string, noColor bool) Theme {
	t, ok := themeRegistry[name]
	if !ok {
		t = themeRegistry["default"]
	}
	if noColor {
		t.UseColor = false
	}
	return t
}

func init() {
	RegisterTheme(Theme{
		Name: "default", UseColor: true, UseIcons: true,
		Cursor: "❯", Checked: "◉", Unchecked: "○",
	})
	RegisterTheme(Theme{
		Name: "minimal", UseColor: false, UseIcons: false,
		Cursor: ">", Checked: "[x]", Unchecked: "[ ]",
	})
}

func (t Theme) color(code, s string) string {
	if !t.UseColor {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

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
	return t.color("1;36", msg)
}

// Dim formats de-emphasized text (e.g. "(coming soon)" hints).
func (t Theme) Dim(msg string) string {
	return t.color("2", msg)
}
