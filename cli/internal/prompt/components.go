package prompt

import (
	"fmt"
	"strings"
	"time"
)

// components.go — the reusable UI component library (design-system §3.1).
// Every function returns []string of fully styled lines; components never
// write directly. Screens and widgets compose these.

// BrandHeader returns the Lumo brand wordmark + tagline (M-01 Brand).
// One line, no ASCII art (design-spec §1.3).
func BrandHeader(t Theme) string {
	return t.Primary("Lumo") + t.Dim(" — a new project, ready in seconds")
}

// SectionTitle returns a section header line (M-01 Section).
func SectionTitle(t Theme, text string) string {
	return t.Header(text)
}

// Caption returns a muted single line (A-03 Caption).
func Caption(t Theme, text string) string {
	return t.Muted(text)
}

// DividerLine returns a horizontal rule at the given width (A-04 Divider).
func DividerLine(t Theme, width int) string {
	if width <= 0 {
		width = LayoutMaxWidth
	}
	return t.Border(strings.Repeat(t.Panel[PanelHorizontal], width))
}

// PanelRow formats a label: value pair for panels (M-09 StatusPanel).
// Label is Dim, value is default.
func PanelRow(t Theme, label, value string) string {
	return t.Dim(label) + value
}

// StatusPanel builds a bordered panel from an optional Accent title and
// rows (each a label: value pair already formatted by PanelRow). M-09.
func StatusPanel(t Theme, title string, rows []string) []string {
	l := LayoutEngine{}
	return l.Panel(t, title, rows, 0)
}

// ConfirmCard builds the confirmation panel (M-10 ConfirmCard): bordered
// box with label: value rows and a blank footer line.
func ConfirmCard(t Theme, title string, rows []string) []string {
	l := LayoutEngine{}
	return l.Panel(t, title, rows, 0)
}

// Key represents one binding in the hint bar (M-08 HintBar).
type Key struct {
	Label string   // e.g. "move", "select"
	Keys  []string // e.g. ["↑", "↓", "j", "k"]
}

// KeyHints formats the footer hint bar (M-08 HintBar): a single dim
// line joining "Keys Label" with " · " separators. When width < 40 or
// the output isn't a TTY, returns "" so the bar is hidden
// (design-system §2.3).
func KeyHints(t Theme, keys []Key, width int, tty bool) string {
	if !tty || width < 40 {
		return ""
	}
	var parts []string
	for _, k := range keys {
		keysStr := strings.Join(k.Keys, "/")
		parts = append(parts, keysStr+" "+k.Label)
	}
	return t.Muted(strings.Join(parts, " · "))
}

// FileTree renders the generated artifact tree (M-12 FileTree).
// Default theme: first N-1 rows "├─ file", last "└─ file", no icons: "+ file".
func FileTree(t Theme, files []string) []string {
	if len(files) == 0 {
		return nil
	}
	plus := t.Glyph(GlyphTreePlus)
	mid := t.Glyph(GlyphTreeMid)
	end := t.Glyph(GlyphTreeEnd)
	var lines []string
	for i, f := range files {
		var glyph string
		if i == len(files)-1 {
			glyph = end
		} else {
			glyph = mid
		}
		if !t.UseIcons {
			glyph = plus
		}
		lines = append(lines, t.Info("  "+glyph+" "+f))
	}
	return lines
}

// NextSteps renders the follow-up list (M-11 NextSteps).
// Default theme: "→ step"; minimal: "- step", Accent arrow.
func NextSteps(t Theme, steps []string) []string {
	if len(steps) == 0 {
		return nil
	}
	arrow := t.Accent(t.Glyph(GlyphArrow))
	var lines []string
	lines = append(lines, t.Header("Next steps:"))
	for _, s := range steps {
		lines = append(lines, "  "+arrow+" "+t.Dim(s))
	}
	return lines
}

// NoteLine formats an inline advisory (M-13 Note): "Note: " in Warn +
// text. Used for dry-run, slow-phase timing, capability dependency
// notes.
func NoteLine(t Theme, text string) string {
	return t.Warn("Note: ") + text
}

// ErrorPanel builds the failure screen (O-04 Failure, design-system
// §4 "S09: ErrorPanel anatomy"): a "[FAIL]/✗ <message>" line (Failure),
// an indented hint (Warn), and an optional indented docs reference
// (Dim) — deliberately unbordered, unlike StatusPanel/ConfirmCard: the
// spec composes O-04 from Failure + Note, not a panel box. hint and ref
// may be "" to omit either line.
func ErrorPanel(t Theme, headline, hint, ref string) []string {
	lines := []string{"", t.Failure(headline)}
	if hint != "" {
		lines = append(lines, t.Warn("  "+hint))
	}
	if ref != "" {
		lines = append(lines, t.Dim("  reference: "+ref))
	}
	return append(lines, "")
}

// ProgressBar renders the percentage bar (M-04 ProgressBar).
// Default: "[███████░░░░] 60%"; minimal: "[##########--------] 60%".
func ProgressBar(t Theme, pct int, width int) string {
	if width <= 0 {
		width = 24
	}
	if !t.UseIcons {
		width = 20
	}
	full, empty := "█", "░"
	if !t.UseIcons {
		full, empty = "#", "-"
	}
	filled := (width * pct) / 100
	if filled > width {
		filled = width
	}
	bar := strings.Repeat(full, filled) + strings.Repeat(empty, width-filled)
	return fmt.Sprintf("[%s] %d%%", bar, pct)
}

// SummaryLine builds the plan line (M-02 SummaryLine):
// "→ Creating my-app — backend-service · go · rest-api · git-init"
// Arrow in Accent, name default, rest Dim/Muted.
func SummaryLine(t Theme, plan string) string {
	return t.Accent(t.Glyph(GlyphArrow)) + " " + plan
}

// SpinnerLine builds a single animated step line: frame + label.
// Default: frame in Primary, label Bold; minimal: frame plain.
func SpinnerLine(t Theme, frame, label string, isActive bool) string {
	if isActive {
		return "  " + t.Primary(frame) + " " + t.Bold(label)
	}
	return "  " + frame + " " + label
}

// formatDuration renders a duration with one decimal, e.g. 920ms → "0.9s".
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.1fs", d.Seconds())
}
