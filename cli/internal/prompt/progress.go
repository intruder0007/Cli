package prompt

import (
	"io"
	"math"
	"strings"
	"sync"
	"time"
)

// progress.go — the persistent step-based progress engine
// (docs/cli/design-system.md §5). Steps come from the engine's real
// Progress callbacks, never guessed: the template phase plus one
// "applying capability <id>" per selected capability, in execution
// order. Each step is pending → active → done | failed; rows are never
// overwritten on a terminal — only the active row's spinner advances,
// and the percentage bar refreshes on phase transitions. No percentage
// is faked: the bar is done/total where total = 1 + len(capabilities)
// is known up front.
//
// On a terminal this renders as the generation dashboard: the step
// stack, the percentage bar, a live status line (tasks completed /
// remaining, elapsed time, current stage), and the slow-phase note.
// Off a terminal it degrades to one static line per phase — the arrow
// line as the phase begins, overtyped in place with the final glyph
// when it completes — so piped/CI transcripts are deterministic and
// contain no escape codes.
type ProgressGroup struct {
	w     *Output
	t     Theme
	tty   bool
	total int

	mu          sync.Mutex
	slots       []*progressStep // slot 0 = template, then one per capability
	done        int
	start       time.Time
	started     bool
	noteShown   bool
	activeFrame string
	animates    bool

	region   *Region
	animStop chan struct{}
	animDone chan struct{}
}

type stepState int

const (
	stepPending stepState = iota
	stepActive
	stepDone
	stepFailed
)

type progressStep struct {
	label   string
	state   stepState
	started time.Time
	elapsed time.Duration
}

// NewProgressGroup returns a ProgressGroup writing to w, animating only
// if w is a terminal and the theme uses icons (design-spec §8.2:
// --no-color/minimal = no motion). caps is the selected capability ids
// in selection order; the bar's total is 1 + len(unique caps).
// Capability rows are reserved up front so pending steps are visible
// before they begin; the template row appears when its phase starts.
func NewProgressGroup(w io.Writer, t Theme, caps []string) *ProgressGroup {
	o := NewOutput(w)
	g := &ProgressGroup{t: t, tty: o.TTY(), region: NewRegion(o)}
	g.w = o
	seen := make(map[string]bool, len(caps))
	for _, c := range caps {
		if seen[c] {
			continue
		}
		seen[c] = true
		g.slots = append(g.slots, &progressStep{label: "applying capability " + c, state: stepPending})
	}
	g.total = 1 + len(g.slots)
	g.animates = g.tty && t.UseIcons
	return g
}

// Start marks the phase as active. The engine emits one Start per phase
// (done=false) in execution order; the first Start primes the bar and
// starts the animation on a terminal.
func (g *ProgressGroup) Start(phase string) {
	g.mu.Lock()
	if !g.started {
		g.started = true
		g.start = time.Now()
	}
	g.markActiveLocked(phase)
	if !g.tty {
		g.mu.Unlock()
		g.printLine(g.t.Accent(g.arrow()), phase)
		return
	}
	if g.animates && g.animStop == nil {
		g.animStop = make(chan struct{})
		g.animDone = make(chan struct{})
		go g.animate()
	}
	g.renderLocked()
	g.mu.Unlock()
}

// Done marks the phase as successfully completed (engine done=true),
// recording its real duration. On a terminal the row is finalized in
// place; off a terminal the phase's arrow line is overtyped with the
// done glyph (and timing when the phase took at least 0.1s).
func (g *ProgressGroup) Done(phase string) {
	g.mu.Lock()
	st := g.findSlotLocked(phase)
	if st == nil {
		st = &progressStep{label: phase, state: stepPending}
		g.slots = append(g.slots, st)
	}
	if st.state != stepDone {
		if st.state == stepActive {
			st.elapsed = time.Since(st.started)
		}
		st.state = stepDone
		g.done++
	}
	if !g.tty {
		g.mu.Unlock()
		g.printOvertype(g.colorGreen(g.doneGlyph()), phaseWithTiming(st))
		return
	}
	g.renderLocked()
	g.mu.Unlock()
}

// Fail freezes the bar with the in-flight phase marked failed. The
// engine reports a failed phase's start but never its completion, so
// the error return is the failure signal (design-spec §5.3.3). Pending
// rows stay frozen below the error screen, showing what never ran.
func (g *ProgressGroup) Fail() {
	g.mu.Lock()
	var st *progressStep
	for i := len(g.slots) - 1; i >= 0; i-- {
		if g.slots[i].state == stepActive {
			st = g.slots[i]
			break
		}
	}
	if st != nil {
		st.state = stepFailed
		st.elapsed = time.Since(st.started)
	}
	if !g.tty {
		g.mu.Unlock()
		if st != nil {
			g.printOvertype(g.colorRed(g.failGlyph()), st.label)
		}
		return
	}
	g.stopAnimationLocked()
	g.renderLocked()
	g.mu.Unlock()
}

// Finish stops the animation and returns the total generation time
// (from the first phase's start to now), for the completion screen.
func (g *ProgressGroup) Finish() time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.stopAnimationLocked()
	if !g.started {
		return 0
	}
	return time.Since(g.start)
}

// Percent returns the real completed fraction (0-100) from done/total.
func (g *ProgressGroup) Percent() int {
	if g.total == 0 {
		return 0
	}
	return int(math.Round(100 * float64(g.done) / float64(g.total)))
}

type progressRender struct {
	lines      []string
	statusLine string
}

// markActiveLocked transitions the phase's slot to active, creating the
// slot if the phase wasn't reserved (the template phase). A template
// phase always lands in slot 0; an unknown capability phase appends.
func (g *ProgressGroup) markActiveLocked(phase string) {
	st := g.findSlotLocked(phase)
	if st == nil {
		st = &progressStep{label: phase, state: stepPending}
		if strings.HasPrefix(phase, "generating template") {
			g.slots = append([]*progressStep{st}, g.slots...)
		} else {
			g.slots = append(g.slots, st)
		}
	}
	st.state = stepActive
	st.started = time.Now()
}

func (g *ProgressGroup) findSlotLocked(phase string) *progressStep {
	for _, s := range g.slots {
		if s.label == phase {
			return s
		}
	}
	return nil
}

// renderLocked redraws the whole dashboard in place: step rows,
// percentage bar, status line, and the slow-phase note if it has
// appeared. The first render writes from the current cursor position
// (just below the plan line); later renders move up to the stack's top
// and clear down, so the plan line above is never touched.
func (g *ProgressGroup) renderLocked() {
	g.region.Draw(g.buildLinesLocked(framesFor(g.t)))
}

// buildLinesLocked assembles the dashboard frame. The status line and
// slow note are terminal-only; the step rows and the percentage bar are
// common to both themes (and to the pinned test formats).
func (g *ProgressGroup) buildLinesLocked(frames []string) []string {
	lines := []string{}
	for _, s := range g.slots {
		lines = append(lines, g.writeStep(s, frames))
	}
	lines = append(lines, g.writeBar())
	if g.animates {
		lines = append(lines, g.writeStatus())
	}
	if g.noteShown {
		lines = append(lines, g.t.Warn("Note: ")+"this phase is slower than usual — hanging?")
	}
	return lines
}

func framesFor(t Theme) []string {
	if len(t.Spinner) > 0 {
		return t.Spinner
	}
	return []string{"|", "/", "-", "\\"}
}

func (g *ProgressGroup) writeStep(s *progressStep, frames []string) string {
	switch s.state {
	case stepPending:
		return "  " + g.t.Muted(g.pendingGlyph()) + " " + s.label
	case stepActive:
		frame := g.activeFrame
		if frame == "" {
			frame = frames[0]
		}
		line := "  " + g.t.Primary(frame) + " " + g.t.Bold(s.label)
		if el := time.Since(s.started); el >= 2*time.Second {
			line += " " + g.t.Dim("("+formatDuration(el)+")")
			if !g.noteShown && el >= 3*time.Second {
				g.noteShown = true
			}
		}
		return line
	case stepDone:
		line := "  " + g.colorGreen(g.doneGlyph()) + " " + s.label
		if s.elapsed >= 100*time.Millisecond {
			line += " " + g.t.Dim("("+formatDuration(s.elapsed)+")")
		}
		return line
	case stepFailed:
		return "  " + g.colorRed(g.failGlyph()) + " " + g.t.Bold(s.label)
	}
	return ""
}

func (g *ProgressGroup) writeBar() string {
	return ProgressBar(g.t, g.Percent(), 0)
}

// writeStatus renders the live dashboard status line; it exists only
// on an animating terminal. It carries no information that the text
// rows don't (percent, elapsed) but surfaces it on one quiet line.
func (g *ProgressGroup) writeStatus() string {
	el := time.Since(g.start)
	done := g.done
	total := g.total
	stage := ""
	for _, s := range g.slots {
		if s.state == stepActive {
			stage = s.label
			break
		}
	}
	status := "tasks " + itoa(done) + "/" + itoa(total) + " · elapsed " + formatDuration(el)
	if stage != "" {
		status += " · stage: " + stage
	}
	return "  " + g.t.Muted(status)
}

func (g *ProgressGroup) animate() {
	frames := framesFor(g.t)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	var frameIdx int
	lastFrame := time.Now()
	for {
		select {
		case <-g.animStop:
			close(g.animDone)
			return
		case <-tick.C:
			g.mu.Lock()
			now := time.Now()
			if now.Sub(lastFrame) >= 700*time.Millisecond { // design §2.5 cadence
				lastFrame = now
				frameIdx++
				g.activeFrame = frames[frameIdx%len(frames)]
			}
			g.renderLocked()
			g.mu.Unlock()
		}
	}
}

func (g *ProgressGroup) stopAnimationLocked() {
	if g.animStop == nil {
		return
	}
	close(g.animStop)
	<-g.animDone
	g.animStop = nil
}

// printLine writes a fresh off-terminal line for a phase beginning.
func (g *ProgressGroup) printLine(glyph, label string) {
	g.w.Println("  " + glyph + " " + label)
}

// printOvertype rewrites the previous line off a terminal (a bare \r,
// no escape codes) with the phase's final glyph.
func (g *ProgressGroup) printOvertype(glyph, label string) {
	g.w.Println("\r  " + glyph + " " + label)
}

// The color-conditional glyph helpers: on a terminal these carry the
// success/failure color; the text form is identical in both themes.
func (g *ProgressGroup) colorGreen(s string) string { return g.t.Token(TokenSuccess, s) }
func (g *ProgressGroup) colorRed(s string) string   { return g.t.Token(TokenFailure, s) }

func (g *ProgressGroup) arrow() string {
	return g.t.Glyph(GlyphArrow) // "→" on icon themes, "-" under no-color/minimal
}

func (g *ProgressGroup) doneGlyph() string {
	return g.t.Glyph(GlyphDone)
}

func (g *ProgressGroup) failGlyph() string {
	return g.t.Glyph(GlyphFailed)
}

func (g *ProgressGroup) pendingGlyph() string {
	return g.t.Glyph(GlyphPending)
}

// phaseWithTiming appends the step's real duration when it took at
// least 0.1s (faster phases get no timing — "(0.0s)" would be noise).
func phaseWithTiming(s *progressStep) string {
	if s.elapsed < 100*time.Millisecond {
		return s.label
	}
	return s.label + " (" + formatDuration(s.elapsed) + ")"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
