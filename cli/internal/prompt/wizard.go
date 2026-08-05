package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/intruder0007/Lumo/core/config"
)

// wizard.go — the interactive wizard (organism O-01). On a real
// terminal it runs as a sequence of single-focus screens:
//
//	welcome → name → location → theme → type → language → framework
//	        → capabilities (when installed) → confirm → (generate)
//
// Each screen is one widget (input / menu / multi-select / confirm),
// re-rendered atomically (design-spec §3.1). Every step carries a
// title ("Step N of M · <name>", the progress indicator), the body
// widget, and a contextual hint bar — answering "where am I", "what is
// happening", "what can I do", and "what happens next".
//
// The location step always asks where to generate — never the current
// working directory (which, on Windows, is the launching .exe's own
// folder when double-clicked) and never silently. It's pre-filled with
// the last-used location (persisted config) or a home-relative default,
// editable every run, and the choice is re-persisted on each run that
// reaches it.
//
// Navigation (design-spec §3.2/3.3): ↑↓/jk move, Home/End or g/G jump,
// Enter confirms, Space toggles, Esc clears a filter first then goes
// back, ←/h also go back, Ctrl+C/q cancel. Back navigation preserves
// the answers already collected. When stdin/stdout aren't both real
// terminals, RunWizard falls back to plain numbered prompts
// (runWizardLine) — deterministic and piped-safe.

// noTemplatesErr is returned by both wizard paths when discovery found
// no template plugins: nothing the user could pick would resolve, so
// the wizard fails fast instead of asking dead-end questions.
func noTemplatesErr() error {
	return errors.New("no template plugins found — run `lumo plugins list` to inspect discovery, and check LUMO_PLUGIN_DIRS if templates/ isn't next to the binary")
}

// Wizard step numbers. Confirm is always last; capabilities sits before
// it only when at least one capability plugin is installed, so the
// step count (and the "N of M" progress indicator) reflects reality.
const (
	stepName     = 1
	stepLocation = 2
	stepTheme    = 3
	stepType     = 4
	stepLang     = 5
	stepFw       = 6
)

// defaultProjectsDir is the location step's fallback pre-fill when no
// location has been persisted yet (first run): the user's home
// directory plus "Projects" — stable regardless of how the binary was
// launched, unlike the current working directory. "." is the last
// resort if the home directory can't be determined.
func defaultProjectsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "Projects")
}

// persistProjectsDir saves the interactively-chosen project location
// (wizard-only, mirroring persistTheme — --dir/--answers never write
// it).
func persistProjectsDir(dir string) {
	cfg, _ := LoadConfig()
	cfg.DefaultProjectsDir = dir
	_ = SaveConfig(cfg)
}

// RunWizard walks the full interactive flow on a real terminal, or the
// numbered-prompt fallback otherwise. version is shown on the welcome
// screen.
func RunWizard(out io.Writer, spec WizardSpec, version string) (config.Answers, error) {
	stdinFd := int(os.Stdin.Fd())
	stdoutFd := int(os.Stdout.Fd())
	if term.IsTerminal(stdinFd) && term.IsTerminal(stdoutFd) {
		a, err := runWizardTUI(stdinFd, out, spec, version)
		switch err {
		case nil, ErrCancelled:
			return a, err
		default:
			// Raw mode failed to engage, or some other terminal-level
			// problem — degrade to the line-based wizard.
			fmt.Fprintln(out, "(falling back to plain input)")
		}
	}
	return runWizardLine(bufio.NewReader(os.Stdin), out, spec)
}

// wizardSteps returns the total step count for this spec.
func wizardSteps(spec WizardSpec) int {
	steps := 7 // name, location, theme, type, language, framework, confirm
	if len(spec.Capabilities) > 0 {
		steps = 8
	}
	return steps
}

// wizardStepTitle builds the step chrome (see chrome.go) for step st of
// a steps-step wizard run.
func wizardStepTitle(st, steps int) string {
	return StepChrome(st, steps, wizardStepName(st))
}

// runWizardTUI runs the terminal wizard. It puts the terminal into raw
// mode for its whole duration (not per-question) to avoid flicker, and
// guarantees the terminal is restored — including on Ctrl+C (which in
// raw mode arrives as a plain byte rather than a SIGINT) and on an
// external interrupt signal. Notify is registered before MakeRaw so no
// window exists in which an interrupt can't restore the terminal.
func runWizardTUI(fd int, out io.Writer, spec WizardSpec, version string) (config.Answers, error) {
	var a config.Answers

	if len(spec.Templates) == 0 {
		return a, noTemplatesErr()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	raw, err := EnterRaw(fd)
	if err != nil {
		return a, err
	}
	defer raw.Close()
	go func() {
		if _, ok := <-sigCh; ok {
			raw.Close()
			os.Exit(130)
		}
	}()

	in := os.Stdin
	o := NewOutput(out)
	o.HideCursor()
	defer o.ShowCursor()

	// The wizard's starting theme follows the documented precedence
	// (design-spec §10.3): LUMO_THEME overrides the persisted config, and
	// NO_COLOR forces the ASCII rendition regardless of theme.
	cfg, _ := LoadConfig()
	themeID := ResolveThemeName("", cfg.Theme)
	theme := GetTheme(themeID, os.Getenv("NO_COLOR") != "")

	if err := welcomeStep(o, in, theme, version); err != nil {
		// Backing out of the welcome screen leaves the wizard entirely
		// (its ErrBack maps to a cancellation below).
		return a, err
	}

	steps := wizardSteps(spec)
	hasCaps := len(spec.Capabilities) > 0

	// location is collected separately from a.ProjectName (the bare
	// name) and only combined into a path at confirm/return time — see
	// combinedTarget — so backing up to the name or location step never
	// double-joins an already-combined value.
	location := cfg.DefaultProjectsDir

	st := stepName
	for {
		var err error
		switch st {
		case stepName:
			var name string
			name, err = textInput(o, in, theme, wizardStepTitle(st, steps), defaultNamePlaceholder, a.ProjectName)
			if err == nil {
				a.ProjectName = strings.TrimSpace(name)
				if a.ProjectName == "" {
					err = fmt.Errorf("project name is required")
				}
			}
		case stepLocation:
			def := location
			if def == "" {
				def = defaultProjectsDir()
			}
			var loc string
			loc, err = textInput(o, in, theme, wizardStepTitle(st, steps), "", def)
			if err == nil {
				loc = strings.TrimSpace(loc)
				if loc == "" {
					err = fmt.Errorf("project location is required")
				} else {
					location = loc
					persistProjectsDir(loc)
				}
			}
		case stepTheme:
			var id string
			id, err = selectMenu(o, in, theme, wizardStepTitle(st, steps), themeOptions, themeID)
			if err == nil {
				themeID = id
				a.Theme = id
				theme = GetTheme(id, os.Getenv("NO_COLOR") != "")
				persistTheme(id)
			}
		case stepType:
			opts := spec.projectTypeOptions()
			a.ProjectType, err = selectMenu(o, in, theme, wizardStepTitle(st, steps), opts, spec.defaultFor(opts, "backend-service"))
		case stepLang:
			opts := spec.languageOptions(a.ProjectType)
			a.Language, err = selectMenu(o, in, theme, wizardStepTitle(st, steps), opts, spec.defaultFor(opts, "go"))
		case stepFw:
			opts := spec.frameworkOptions(a.ProjectType, a.Language)
			a.Framework, err = selectMenu(o, in, theme, wizardStepTitle(st, steps), opts, spec.defaultFor(opts, "rest-api"))
		case 7:
			if hasCaps {
				var sel []string
				sel, err = multiSelectMenu(o, in, theme, wizardStepTitle(st, steps), spec.capabilityOptions())
				if err == nil {
					a.Capabilities = sel
				}
			} else {
				// no capability plugins: step 7 is the confirm screen.
				err = confirmStep(o, in, theme, st, steps, withTarget(a, location))
			}
		case 8:
			err = confirmStep(o, in, theme, st, steps, withTarget(a, location))
		default:
			return a, fmt.Errorf("wizard: unknown step %d", st)
		}

		switch {
		case errors.Is(err, errConfirmed):
			a.ProjectName = combinedTarget(location, a.ProjectName)
			return a, a.ValidateShape()
		case err == nil:
			st++
		case errors.Is(err, ErrBack):
			if st <= 1 {
				return a, ErrCancelled
			}
			st--
		default:
			return a, err
		}
	}
}

// combinedTarget joins the wizard's separately-collected location and
// bare project name into the single path config.Answers.ProjectName
// carries downstream (see cli/main.go's resolveTargetPath, which
// already knows how to split a path-like ProjectName back into a
// target directory and name — the same mechanism --answers files use).
func combinedTarget(location, name string) string {
	if location == "" {
		return name
	}
	return filepath.Join(location, name)
}

// withTarget returns a copy of a with ProjectName resolved to the
// combined location+name path, for screens (the confirm card) that need
// to display the real target before the wizard has actually returned.
func withTarget(a config.Answers, location string) config.Answers {
	a.ProjectName = combinedTarget(location, a.ProjectName)
	return a
}

// errConfirmed is the internal signal that the user confirmed the
// summary (Enter on the confirm screen): step 7/8 then return the
// answers rather than advancing past the last step (which would fall
// through into the "unknown step" error).
var errConfirmed = errors.New("wizard: confirmed")

// confirmStep runs one confirm-screen iteration: Enter proceeds (via
// errConfirmed), a back key (← / h) returns ErrBack so the outer loop
// walks back a step, and cancel (Ctrl+C/q) reports ErrCancelled.
func confirmStep(o *Output, in io.Reader, t Theme, st, steps int, a config.Answers) error {
	goAhead, err := confirm(o, in, t, wizardStepTitle(st, steps), confirmCard(t, a))
	if err != nil {
		return err
	}
	if goAhead {
		return errConfirmed
	}
	return ErrBack
}

// defaultNamePlaceholder is shown inside the name input while it's empty
// (design-spec §4.2). The location step (stepLocation) collects where
// to generate; this field is just the project's name.
const defaultNamePlaceholder = "my-app"

// persistTheme saves the interactively-chosen theme (wizard-only, per
// ADR-0007 — theme/--answers never write it).
func persistTheme(id string) {
	cfg, _ := LoadConfig()
	cfg.Theme = id
	_ = SaveConfig(cfg)
}

// confirmCard builds the confirmation screen's rows (M1-1 ConfirmCard):
// the resolved target, the stack, and any capabilities.
func confirmCard(t Theme, a config.Answers) []string {
	rows := []string{PanelRow(t, "Target:  ", targetDisplay(a.ProjectName))}
	rows = append(rows, PanelRow(t, "Stack:   ", a.ProjectType+" · "+a.Language+" · "+a.Framework))
	if len(a.Capabilities) > 0 {
		rows = append(rows, PanelRow(t, "Extras:  ", strings.Join(a.Capabilities, ", ")))
	}
	return rows
}

// targetDisplay returns the confirmation screen's rendering of the
// project location: an explicit path (relative/absolute/~) shows the
// resolved absolute target — the exact directory that will be created —
// while a bare name shows the name itself, since its target is simply
// the CWD (design-spec §2.2 rule 4).
func targetDisplay(input string) string {
	if !IsPathLike(input) {
		return input
	}
	dir, _, err := ResolveTargetPath(input)
	if err != nil {
		return input
	}
	return dir
}

// welcomeStep renders the landing screen and waits for Enter to begin.
func welcomeStep(o *Output, r io.Reader, t Theme, version string) error {
	welcomeRender(o, t, version)
	r = &pushbackReader{r: r}
	for {
		k, b, err := readKeyByte(r)
		if err != nil {
			return err
		}
		switch k {
		case keyEnter, keySpace:
			return nil
		case keyCancel:
			if b == 3 || b == 'q' {
				return ErrCancelled
			}
		}
	}
}

// welcomeRender draws the landing screen for the interactive wizard.
func welcomeRender(o *Output, t Theme, version string) {
	o.ClearScreen()
	SetTitle("Lumo — New Project")
	lines := []string{
		BrandHeader(t),
		"",
		Caption(t, "Describe the project — template, language, framework, capabilities —"),
		Caption(t, "and Lumo generates a tested, ready-to-run project in seconds."),
		"",
		t.Dim(fmt.Sprintf("lumo %s", version)),
		"",
	}
	if hint := KeyHints(t, []Key{
		{Label: "start", Keys: []string{"enter"}},
		{Label: "quit", Keys: []string{"ctrl+c"}},
	}, o.Width(), o.TTY()); hint != "" {
		lines = append(lines, hint)
	}
	for _, l := range lines {
		o.Println(l)
	}
}

// ---- Line (piped / non-terminal) fallback -----------------------------

// runWizardLine is the plain-numbered prompts wizard: one line per
// question, used whenever stdin/stdout aren't both real terminals.
// main.go prints the brand banner before calling RunWizard; this path
// adds no header of its own so a piped run shows it exactly once.
// Ctrl+C in cooked mode is a real SIGINT; the handler prints
// "cancelled" and exits 130.
func runWizardLine(in *bufio.Reader, out io.Writer, spec WizardSpec) (config.Answers, error) {
	var a config.Answers

	if len(spec.Templates) == 0 {
		return a, noTemplatesErr()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)
	go func() {
		if _, ok := <-sigCh; ok {
			fmt.Fprintln(out, "cancelled")
			os.Exit(130)
		}
	}()

	a.ProjectName = ask(in, out, "Project name", "")
	if a.ProjectName == "" {
		return a, fmt.Errorf("project name is required")
	}

	cfg, _ := LoadConfig()

	// Location is always asked here too — never defaulted to the
	// current working directory silently — pre-filled with the
	// last-used location or a home-relative default (see
	// defaultProjectsDir), same as the TUI wizard's stepLocation.
	def := cfg.DefaultProjectsDir
	if def == "" {
		def = defaultProjectsDir()
	}
	location := ask(in, out, "Location", def)
	if location == "" {
		return a, fmt.Errorf("project location is required")
	}
	persistProjectsDir(location)

	themeID := cfg.Theme
	if themeID == "" {
		themeID = "default"
	}
	a.Theme = askChoice(in, out, "Theme", themeOptions, themeID)
	persistTheme(a.Theme)

	projectTypes := spec.projectTypeOptions()
	a.ProjectType = askChoice(in, out, "Project type", projectTypes, spec.defaultFor(projectTypes, "backend-service"))
	langOpts := spec.languageOptions(a.ProjectType)
	a.Language = askChoice(in, out, "Language", langOpts, spec.defaultFor(langOpts, "go"))
	fwOpts := spec.frameworkOptions(a.ProjectType, a.Language)
	a.Framework = askChoice(in, out, "Framework", fwOpts, spec.defaultFor(fwOpts, "rest-api"))
	if len(spec.Capabilities) > 0 {
		a.Capabilities = askMulti(in, out, "Capabilities (comma-separated ids, blank for none)", spec.capabilityOptions())
	}
	a.ProjectName = combinedTarget(location, a.ProjectName)
	return a, a.ValidateShape()
}

func ask(in *bufio.Reader, out io.Writer, label, def string) string {
	if def != "" {
		fmt.Fprintf(out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(out, "%s: ", label)
	}
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// askChoice prints numbered options and reprompts until an available
// option is chosen by number or id.
func askChoice(in *bufio.Reader, out io.Writer, label string, opts []option, def string) string {
	for {
		fmt.Fprintf(out, "%s:\n", label)
		for i, o := range opts {
			suffix := ""
			if !o.Available {
				suffix = " (coming soon)"
			}
			fmt.Fprintf(out, "  %d) %s%s\n", i+1, o.Name, suffix)
		}
		answer := ask(in, out, "Choose", def)

		for i, o := range opts {
			if answer == o.ID || answer == fmt.Sprintf("%d", i+1) || strings.EqualFold(answer, o.Name) {
				if !o.Available {
					fmt.Fprintf(out, "%s is coming soon; please choose an available option.\n", o.Name)
					break
				}
				return o.ID
			}
		}
		if answer == def {
			return def
		}
	}
}

func askMulti(in *bufio.Reader, out io.Writer, label string, opts []option) []string {
	for {
		fmt.Fprintf(out, "%s:\n", label)
		for i, o := range opts {
			fmt.Fprintf(out, "  %d) %s — %s\n", i+1, o.Name, o.ID)
		}
		answer := ask(in, out, "Choose", "")
		if answer == "" {
			return nil
		}

		var selected []string
		ok := true
		for _, tok := range strings.Split(answer, ",") {
			tok = strings.TrimSpace(tok)
			matched := ""
			for i, o := range opts {
				if tok == o.ID || tok == fmt.Sprintf("%d", i+1) || strings.EqualFold(tok, o.Name) {
					matched = o.ID
					break
				}
			}
			if matched == "" {
				fmt.Fprintf(out, "unknown capability %q\n", tok)
				ok = false
				break
			}
			selected = append(selected, matched)
		}
		if ok {
			return selected
		}
	}
}
