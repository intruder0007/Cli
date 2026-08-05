package prompt

import (
	"strings"
	"unicode"
)

// Layout constants (docs/cli/design-system.md §2.3).
const (
	// LayoutMaxWidth is the widest any component line renders; layouts
	// beyond it are truncated, never wrapped.
	LayoutMaxWidth = 80
	// LayoutIndent is the base indentation for nested content.
	LayoutIndent = 2
)

// LayoutEngine is the pure-function layer that turns styled strings
// into correctly sized lines: visible-width measurement, truncation,
// padding, alignment, dividers, and box drawing. It knows nothing
// about screens or state — components and widgets call into it.
type LayoutEngine struct {
	// MaxWidth is the terminal width the layout clamps to. 0 means the
	// engine's own default (LayoutMaxWidth).
	MaxWidth int
}

// width returns the effective layout width: the configured width
// clamped to LayoutMaxWidth (design rule: max 80 cols).
func (l LayoutEngine) width() int {
	w := l.MaxWidth
	if w <= 0 || w > LayoutMaxWidth {
		return LayoutMaxWidth
	}
	return w
}

// Width measures the visible width of s, ignoring ANSI styling.
func (l LayoutEngine) Width(s string) int {
	return displayWidth(stripANSI(s))
}

// Truncate shortens s to fit visible width max, appending the ellipsis
// glyph when it had to cut (design-system §2.1). The result never
// exceeds max visible columns.
func (l LayoutEngine) Truncate(s string, max int) string {
	if l.Width(s) <= max {
		return s
	}
	ell := "…"
	if max < 2 {
		return ""
	}
	plain := stripANSI(s)
	// Keep max-1 visible columns plus the ellipsis.
	keep := max - 1
	var b strings.Builder
	cols := 0
	for _, r := range plain {
		w := runeWidth(r)
		if cols+w > keep {
			break
		}
		b.WriteRune(r)
		cols += w
	}
	return b.String() + ell
}

// Pad right-pads s with spaces to exactly w visible columns, or
// truncates when s is wider (used to line up panel rows).
func (l LayoutEngine) Pad(s string, w int) string {
	diff := w - l.Width(s)
	switch {
	case diff < 0:
		return l.Truncate(s, w)
	case diff > 0:
		return s + strings.Repeat(" ", diff)
	}
	return s
}

// Center centers s within w visible columns.
func (l LayoutEngine) Center(s string, w int) string {
	diff := w - l.Width(s)
	if diff <= 0 {
		return s
	}
	left := diff / 2
	return strings.Repeat(" ", left) + s
}

// Rule returns a horizontal divider of the given glyph, at most w
// visible columns wide (A-04 Divider; default: ─, minimal: -).
func (l LayoutEngine) Rule(glyph string, w int) string {
	if w <= 0 {
		w = l.width()
	}
	return strings.Repeat(glyph, w)
}

// Panel renders a bordered box (M-09 StatusPanel / M-10 ConfirmCard)
// using the theme's panel glyphs: an optional Accent title row, label:
// value rows (labels Dim, values default), and frame lines. The width
// derives from the widest row, clamped to the layout width.
func (l LayoutEngine) Panel(t Theme, title string, rows []string, width int) []string {
	if width <= 0 {
		width = l.width() // fall back, tightened below from row widths
	}
	inner := width - 2
	for _, r := range rows {
		if l.Width(r) > inner {
			inner = l.Width(r)
		}
	}
	width = inner + 2
	if width > l.width() {
		width = l.width()
	}
	inner = width - 2

	tl, tr, bl, br := t.Panel[PanelTopLeft], t.Panel[PanelTopRight],
		t.Panel[PanelBottomLeft], t.Panel[PanelBottomRight]
	h, v := t.Panel[PanelHorizontal], t.Panel[PanelVertical]

	top := t.Border(tl) + l.titleSpan(t, title, inner) + t.Border(tr)
	bottom := t.Border(bl + strings.Repeat(h, inner) + br)

	lines := []string{top}
	for _, r := range rows {
		lines = append(lines, t.Border(v)+" "+l.Pad(r, inner)+" "+t.Border(v))
	}
	lines = append(lines, bottom)
	return lines
}

// titleSpan builds the inner part of the top border: the Accent title
// (if any) plus horizontal border fill to the full inner width.
func (l LayoutEngine) titleSpan(t Theme, title string, inner int) string {
	if title == "" {
		return strings.Repeat(t.Panel[PanelHorizontal], inner)
	}
	tw := displayWidth(title)
	remain := inner - tw
	switch {
	case remain <= 0:
		p := l.Truncate(title, inner)
		return t.Token(TokenAccent, p)
	case remain < 3:
		return t.Token(TokenAccent, title) + strings.Repeat(" ", remain)
	default:
		return t.Token(TokenAccent, title) + " " + strings.Repeat(t.Panel[PanelHorizontal], remain-1)
	}
}

// displayWidth measures a plain (unstyled) string in terminal columns:
// most runes are 1, wide/fullwidth runes (CJK, emoji) are 2, combining
// marks 0.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

func runeWidth(r rune) int {
	switch {
	case r == '\t':
		return 1
	case unicode.Is(unicode.Mn, r), unicode.Is(unicode.Me, r): // combining
		return 0
	case r >= 0x1100 && (r <= 0x115F || r == 0x2329 || r == 0x232A ||
		(r >= 0x2E80 && r <= 0xA4CF && r != 0x303F) ||
		(r >= 0xAC00 && r <= 0xD7A3) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0xFE10 && r <= 0xFE19) ||
		(r >= 0xFE30 && r <= 0xFE6F) ||
		(r >= 0xFF00 && r <= 0xFF60) ||
		(r >= 0xFFE0 && r <= 0xFFE6) ||
		(r >= 0x1F300 && r <= 0x1F64F) ||
		(r >= 0x1F900 && r <= 0x1F9FF) ||
		(r >= 0x20000 && r <= 0x2FFFD) ||
		(r >= 0x30000 && r <= 0x3FFFD)):
		return 2
	}
	return 1
}
