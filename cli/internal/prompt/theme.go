// Package prompt implements the Lumo terminal UI: the design-token
// theme engine (theme.go), the rendering engine (engine.go), the layout
// engine (layout.go), the reusable component library (components.go),
// the input handler (input.go), the interactive widgets (widgets.go),
// the persistent progress engine (progress.go), and the screens
// (screens.go, wizard.go) that compose them. It is the only place cli
// talks to a terminal, so the terminal/theme library choice stays
// swappable without touching core. See docs/cli/design-system.md and
// ADR-0007.
package prompt

import (
	"os"
	"strings"
)

// Token names a semantic color role in the Lumo design system
// (docs/cli/design-system.md §2.2). Components tag their text with a
// Token, never with a raw ANSI code; the theme engine resolves the
// token to the active theme's styling.
type Token string

const (
	TokenPrimary Token = "primary" // wordmark, active cursor
	TokenAccent  Token = "accent"  // plan-line prefix, panel headers
	TokenSuccess Token = "success" // completed, confirmations
	TokenFailure Token = "failure" // errors, failed steps
	TokenWarn    Token = "warn"    // advisories, hints
	TokenDim     Token = "dim"     // secondary text, next steps
	TokenMuted   Token = "muted"   // pending steps, captions
	TokenBorder  Token = "border"  // panel frames, dividers
	TokenHeader  Token = "header"  // step titles, section headers
	TokenInfo    Token = "info"    // values in reports
	TokenBold    Token = "bold"    // the active step's label
	TokenCaret   Token = "caret"   // the input caret
)

// tokenCode maps each Token to its ANSI SGR code in a color theme.
// All default-theme pairs are chosen for WCAG AA contrast on the
// common light and dark terminal backgrounds (design-spec §8.4).
var tokenCode = map[Token]string{
	TokenPrimary: "1;36",
	TokenAccent:  "38;5;75",
	TokenSuccess: "32",
	TokenFailure: "31",
	TokenWarn:    "38;5;214",
	TokenDim:     "2",
	TokenMuted:   "2",
	TokenBorder:  "38;5;240",
	TokenHeader:  "1;36",
	TokenInfo:    "37",
	TokenBold:    "1",
	TokenCaret:   "38;5;75",
}

// GlyphKind names one of the theme-aware glyphs (design-system §2.4).
// A glyph is never interpolated into a string: components compose it
// like any other styled segment, and the theme resolves it — the
// minimal theme always gets an ASCII equivalent, so no state exists
// only as a unicode glyph.
type GlyphKind int

const (
	GlyphCursor    GlyphKind = iota // menu cursor
	GlyphChecked                    // multi-select selected
	GlyphUnchecked                  // multi-select unselected
	GlyphDone                       // completed step
	GlyphFailed                     // failed step
	GlyphPending                    // pending step
	GlyphArrow                      // plan line / next steps
	GlyphTreePlus                   // first file tree rows
	GlyphTreeMid                    // middle file tree rows
	GlyphTreeEnd                    // last file tree row
	GlyphCaret                      // text input caret
)

// PanelGlyphKind names one of the box-drawing glyphs (design-system
// §2.3). The minimal theme substitutes ASCII, so panels stay complete
// in both themes.
type PanelGlyphKind int

const (
	PanelTopLeft PanelGlyphKind = iota
	PanelTopRight
	PanelBottomLeft
	PanelBottomRight
	PanelVertical
	PanelHorizontal
)

// Theme is a named set of rendering choices: whether to use color and
// icons, which glyphs the widgets use, the spinner frames, the design
// token palette, and the panel box-drawing characters. A Theme is
// data, not behavior baked into the CLI — see RegisterTheme. The
// semantic color helpers (Primary/Accent/Warn/...) are the design
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
	// Spinner frames animate the in-progress phase of a run
	// (progress.go). Empty means an ASCII fallback is used.
	Spinner []string

	// Palette is the design-token → ANSI SGR map. Empty in the
	// minimal theme (everything renders plain).
	Palette map[Token]string

	// Panel glyphs are the box-drawing characters for StatusPanel /
	// ConfirmCard (design-system §2.3).
	Panel [6]string

	// Glyphs resolves every GlyphKind for this theme.
	Glyphs map[GlyphKind]string
}

var (
	themeRegistry = map[string]Theme{}
	themeOrder    []string
)

// RegisterTheme adds (or replaces) a theme by name. This is the seam a
// future theme-plugin mechanism would call into — see ADR-0007 and
// docs/architecture/roadmap.md. It's an in-process extension point
// today, not a subprocess plugin: a terminal color scheme doesn't need
// the full plugin protocol's isolation guarantees.
func RegisterTheme(t Theme) {
	if _, exists := themeRegistry[t.Name]; !exists {
		themeOrder = append(themeOrder, t.Name)
	}
	themeRegistry[t.Name] = withDefaults(t)
}

// ThemeNames returns registered theme names in registration order.
func ThemeNames() []string {
	out := make([]string, len(themeOrder))
	copy(out, themeOrder)
	return out
}

// GetTheme looks up a theme by name, falling back to "default" if the
// name is unknown or empty. noColor forces both UseColor and UseIcons
// off regardless of the theme's own defaults and swaps the color-cue
// glyphs (❯/◉/▏ and the like) for their ASCII equivalents — a plain
// terminal shouldn't show icon-in-color cues either. minimal is already
// UseColor:false/UseIcons:false, so noColor is a no-op for it.
func GetTheme(name string, noColor bool) Theme {
	t, ok := themeRegistry[name]
	if !ok {
		t = themeRegistry["default"]
	}
	if noColor {
		t.UseColor = false
		t.UseIcons = false
		if m, ok := themeRegistry["minimal"]; ok {
			t.Cursor = m.Cursor
			t.Checked = m.Checked
			t.Unchecked = m.Unchecked
		}
	}
	return t
}

// ResolveThemeName resolves the theme for an interactive run with the
// documented precedence (design-spec §10.3): persisted config first,
// LUMO_THEME overriding it, and an explicit preferred name (the
// --theme flag) overriding both. The preferred name wins whenever it
// names a registered theme; an unknown LUMO_THEME falls back to the
// persisted/default name without failing — the theme registry itself
// falls back to "default" for unknown names.
func ResolveThemeName(preferred, persisted string) string {
	if preferred != "" && isRegisteredTheme(preferred) {
		return preferred
	}
	if env := envOr("LUMO_THEME", ""); env != "" && isRegisteredTheme(env) {
		return env
	}
	if persisted != "" && isRegisteredTheme(persisted) {
		return persisted
	}
	return "default"
}

func isRegisteredTheme(name string) bool {
	_, ok := themeRegistry[name]
	return ok
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func init() {
	RegisterTheme(Theme{
		Name: "default", UseColor: true, UseIcons: true,
		Cursor: "❯", Checked: "◉", Unchecked: "○",
		Spinner: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Palette: map[Token]string{
			TokenPrimary: tokenCode[TokenPrimary], TokenAccent: tokenCode[TokenAccent],
			TokenSuccess: tokenCode[TokenSuccess], TokenFailure: tokenCode[TokenFailure],
			TokenWarn: tokenCode[TokenWarn], TokenDim: tokenCode[TokenDim],
			TokenMuted: tokenCode[TokenMuted], TokenBorder: tokenCode[TokenBorder],
			TokenHeader: tokenCode[TokenHeader], TokenInfo: tokenCode[TokenInfo],
			TokenBold: tokenCode[TokenBold], TokenCaret: tokenCode[TokenCaret],
		},
		Panel: [6]string{"┌", "┐", "└", "┘", "│", "─"},
		Glyphs: map[GlyphKind]string{
			GlyphCursor: "❯", GlyphChecked: "◉", GlyphUnchecked: "○",
			GlyphDone: "✔", GlyphFailed: "✖", GlyphPending: "·",
			GlyphArrow: "→", GlyphTreePlus: "+", GlyphTreeMid: "├─",
			GlyphTreeEnd: "└─", GlyphCaret: "▏",
		},
	})
	RegisterTheme(Theme{
		Name: "minimal", UseColor: false, UseIcons: false,
		Cursor: ">", Checked: "[x]", Unchecked: "[ ]",
		Spinner: []string{"|", "/", "-", "\\"},
		Panel:   [6]string{"+", "+", "+", "+", "|", "-"},
		Glyphs: map[GlyphKind]string{
			GlyphCursor: ">", GlyphChecked: "[x]", GlyphUnchecked: "[ ]",
			GlyphDone: "[x]", GlyphFailed: "[X]", GlyphPending: "-",
			GlyphArrow: ">", GlyphTreePlus: "+", GlyphTreeMid: "+",
			GlyphTreeEnd: "+", GlyphCaret: "_",
		},
	})
}

// withDefaults normalizes a Theme so every optional field has a value:
// a theme registered with only a name and UseColor flags still renders
// glyphs, panels, and tokens. Missing palette/glyph entries resolve to
// the "default" theme's, keeping RegisterTheme's extensibility seam
// cheap to use (design-spec §10.1).
func withDefaults(t Theme) Theme {
	def := themeRegistry["default"]
	if t.Palette == nil {
		t.Palette = map[Token]string{}
		for k, v := range def.Palette {
			t.Palette[k] = v
		}
	}
	if t.Glyphs == nil {
		t.Glyphs = map[GlyphKind]string{}
	}
	for k, v := range def.Glyphs {
		if _, ok := t.Glyphs[k]; !ok {
			t.Glyphs[k] = v
		}
	}
	for i, g := range def.Panel {
		if t.Panel[i] == "" {
			t.Panel[i] = g
		}
	}
	if t.Cursor == "" {
		t.Cursor = def.Cursor
	}
	if t.Checked == "" {
		t.Checked = def.Checked
	}
	if t.Unchecked == "" {
		t.Unchecked = def.Unchecked
	}
	if len(t.Spinner) == 0 {
		t.Spinner = def.Spinner
	}
	return t
}

// color wraps s in the ANSI SGR code for code when the theme uses
// color, returning s unchanged otherwise.
func (t Theme) color(code, s string) string {
	if !t.UseColor || code == "" {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

// Token renders s in the given design token. Components style text
// through this, never through raw ANSI. A bare Theme built by hand
// (nil Palette) falls back to the default token codes, so the semantic
// helpers work on any Theme value; a registered theme with an explicit
// empty palette (the minimal theme) renders plain.
func (t Theme) Token(kind Token, s string) string {
	var code string
	if t.Palette != nil {
		code = t.Palette[kind]
	} else {
		code = tokenCode[kind]
	}
	return t.color(code, s)
}

// asciiGlyph is the ASCII rendering of every GlyphKind, used whenever a
// theme renders without icons (the minimal theme and NO_COLOR/--no-color
// forced runs). A glyph is a semantic cue; when the terminal can't show
// color-coded icons, it degrades to an unambiguous ASCII form instead of
// being dropped or left as a half-rendered unicode character.
var asciiGlyph = map[GlyphKind]string{
	GlyphCursor: ">", GlyphChecked: "[x]", GlyphUnchecked: "[ ]",
	GlyphDone: "[x]", GlyphFailed: "[X]", GlyphPending: "-",
	GlyphArrow: "-", GlyphTreePlus: "+", GlyphTreeMid: "+",
	GlyphTreeEnd: "+", GlyphCaret: "_",
}

// Glyph resolves one theme-aware glyph (design-system §2.4). Icon themes
// (UseIcons) get their registered glyph; iconless rendering (minimal,
// NO_COLOR/--no-color) gets the ASCII equivalent, so no state exists
// only as a unicode glyph.
func (t Theme) Glyph(k GlyphKind) string {
	if !t.UseIcons {
		if g, ok := asciiGlyph[k]; ok {
			return g
		}
	}
	return t.Glyphs[k]
}

// SpinnerFrame returns frame i of the theme's spinner, falling back to
// an ASCII spinner when the theme has none.
func (t Theme) SpinnerFrame(i int) string {
	frames := t.Spinner
	if len(frames) == 0 {
		frames = []string{"|", "/", "-", "\\"}
	}
	if len(frames) == 0 {
		return " "
	}
	return frames[i%len(frames)]
}

// Primary formats text in the brand color — the Lumo wordmark, section
// headers, and the wizard's header lines.
func (t Theme) Primary(s string) string { return t.Token(TokenPrimary, s) }

// Accent highlights interactive elements — progress arrows, next steps.
func (t Theme) Accent(s string) string { return t.Token(TokenAccent, s) }

// Warn flags a non-fatal condition that still deserves attention.
func (t Theme) Warn(s string) string { return t.Token(TokenWarn, s) }

// Border renders structural glyphs (tree branches, dividers) in a
// quieter gray than the surrounding text.
func (t Theme) Border(s string) string { return t.Token(TokenBorder, s) }

// Success formats a success line. Every state has a text label — color
// and icons are additive, never the only signal.
func (t Theme) Success(msg string) string {
	label := "[OK] "
	if t.UseIcons {
		label = "✔ "
	}
	return t.Token(TokenSuccess, label) + msg
}

// Failure formats a failure line.
func (t Theme) Failure(msg string) string {
	label := "[FAIL] "
	if t.UseIcons {
		label = "✗ "
	}
	return t.Token(TokenFailure, label) + msg
}

// Info formats an informational line (e.g. a generated file).
func (t Theme) Info(msg string) string {
	return msg
}

// Header formats a section header.
func (t Theme) Header(msg string) string {
	return t.Token(TokenHeader, msg)
}

// Dim formats de-emphasized text (e.g. "(coming soon)" hints).
func (t Theme) Dim(msg string) string {
	return t.Token(TokenDim, msg)
}

// Muted formats pending/caption text (e.g. pending progress steps).
// Visually identical to Dim today; kept as a separate token so the
// pending-step look can diverge without touching caption styling.
func (t Theme) Muted(msg string) string {
	return t.Token(TokenMuted, msg)
}

// Bold formats text in bold (e.g. the active progress step's label).
func (t Theme) Bold(msg string) string {
	return t.Token(TokenBold, msg)
}

// elide ANSI SGR sequences from s so visible-width math ignores styling.
func stripANSI(s string) string {
	if !strings.ContainsRune(s, '\x1b') {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			// Skip the escape sequence: ESC [ ... m (SGR), or any
			// CSI/OSC until its terminator.
			i++
			for i < len(s) && !(s[i] >= 'a' && s[i] <= 'z') && !(s[i] >= 'A' && s[i] <= 'Z') && s[i] != '\a' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
