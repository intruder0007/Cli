package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"golang.org/x/term"

	"github.com/intruder0007/Lumo/core/config"
)

// noTemplatesErr is returned by both wizard paths when discovery found
// no template plugins: nothing the user could pick would resolve, so
// the wizard fails fast instead of asking dead-end questions.
func noTemplatesErr() error {
	return errors.New("no template plugins found — run `lumo plugins list` to inspect discovery, and check LUMO_PLUGIN_DIRS if templates/ isn't next to the binary")
}

// RunWizard walks through project name, theme, project type, language,
// framework, and capabilities, in that order, writing prompts to out.
// Every menu is built from spec — the discovered template/capability
// plugins — so the wizard only offers what's installed, and the
// framework step is filtered by the chosen project type and language
// (see WizardSpec). When stdin and stdout are both real terminals, it
// uses the arrow-key/space-select menus from menu.go; otherwise (piped
// input, CI, tests) it falls back verbatim to plain numbered-list +
// line input, so no non-interactive usage changes behavior. See
// ADR-0007.
func RunWizard(out io.Writer, spec WizardSpec) (config.Answers, error) {
	stdinFd := int(os.Stdin.Fd())
	stdoutFd := int(os.Stdout.Fd())
	if term.IsTerminal(stdinFd) && term.IsTerminal(stdoutFd) {
		a, err := runWizardTUI(stdinFd, out, spec)
		switch err {
		case nil, ErrCancelled:
			return a, err
		default:
			// Raw mode failed to engage, or some other terminal-level
			// problem — degrade to the line-based wizard rather than
			// fail outright.
			fmt.Fprintln(out, "(falling back to plain input)")
		}
	}
	return runWizardLine(bufio.NewReader(os.Stdin), out, spec)
}

// runWizardTUI runs the arrow-key/space-select wizard. It puts the
// terminal into raw mode for its whole duration (not per-question) to
// avoid flicker, and guarantees the terminal is restored — including on
// Ctrl+C, which in raw mode arrives as a plain byte rather than a real
// SIGINT (see menu.go's readKey), and on an external interrupt signal,
// which is still possible even though the terminal itself won't
// generate one locally. Notify is registered before MakeRaw so no
// window exists in which an interrupt can't restore the terminal.
func runWizardTUI(fd int, out io.Writer, spec WizardSpec) (config.Answers, error) {
	var a config.Answers

	if len(spec.Templates) == 0 {
		return a, noTemplatesErr()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return a, err
	}
	restored := false
	restore := func() {
		if !restored {
			_ = term.Restore(fd, oldState)
			restored = true
		}
	}
	defer restore()

	go func() {
		if _, ok := <-sigCh; ok {
			restore()
			os.Exit(130)
		}
	}()

	in := os.Stdin

	fmt.Fprint(out, "\r\nProject name: ")
	line, cancelled := readLineRaw(in, out)
	a.ProjectName = line
	if cancelled {
		return a, ErrCancelled
	}
	if a.ProjectName == "" {
		return a, fmt.Errorf("project name is required")
	}
	fmt.Fprint(out, "\r\n")

	cfg, _ := LoadConfig()
	defaultTheme := cfg.Theme
	if defaultTheme == "" {
		defaultTheme = "default"
	}

	// The capability step is skipped entirely when no capability
	// plugins are installed, so step numbering is dynamic.
	steps := 4
	if len(spec.Capabilities) > 0 {
		steps = 5
	}
	step := 1

	themeID, err := SelectMenu(out, in, GetTheme(defaultTheme, false), fmt.Sprintf("Step %d of %d · Theme", step, steps), themeOptions, defaultTheme)
	if err != nil {
		return a, err
	}
	step++
	a.Theme = themeID
	t := GetTheme(a.Theme, false)

	// Persist the interactively-chosen theme (wizard-only, per
	// ADR-0007 — --theme/--answers never write this).
	cfg.Theme = a.Theme
	_ = SaveConfig(cfg)

	projectTypes := spec.projectTypeOptions()
	a.ProjectType, err = SelectMenu(out, in, t, fmt.Sprintf("Step %d of %d · Project type", step, steps), projectTypes, spec.defaultFor(projectTypes, "backend-service"))
	if err != nil {
		return a, err
	}
	step++
	langOpts := spec.languageOptions(a.ProjectType)
	a.Language, err = SelectMenu(out, in, t, fmt.Sprintf("Step %d of %d · Language", step, steps), langOpts, spec.defaultFor(langOpts, "go"))
	if err != nil {
		return a, err
	}
	step++
	fwOpts := spec.frameworkOptions(a.ProjectType, a.Language)
	a.Framework, err = SelectMenu(out, in, t, fmt.Sprintf("Step %d of %d · Framework", step, steps), fwOpts, spec.defaultFor(fwOpts, "rest-api"))
	if err != nil {
		return a, err
	}
	if len(spec.Capabilities) > 0 {
		step++
		a.Capabilities, err = MultiSelectMenu(out, in, t, fmt.Sprintf("Step %d of %d · Capabilities (space to toggle, enter to confirm)", step, steps), spec.capabilityOptions())
		if err != nil {
			return a, err
		}
	}

	return a, a.Validate()
}

// readLineRaw reads a line of typed input while the terminal is in raw
// mode (so it must echo each byte itself and handle backspace, unlike
// bufio.Reader.ReadString over a cooked terminal). The bool result
// reports whether the user cancelled with Ctrl+C (raw mode delivers it
// as byte 3 rather than a SIGINT); a cancel inside the project-name
// prompt cancels the whole wizard, matching Ctrl+C on the menus.
func readLineRaw(r io.Reader, w io.Writer) (string, bool) {
	var line []byte
	for {
		b, err := readByte(r)
		if err != nil {
			return string(line), false
		}
		switch b {
		case '\r', '\n':
			return string(line), false
		case 127, 8: // backspace/delete
			if len(line) > 0 {
				line = line[:len(line)-1]
				fmt.Fprint(w, "\b \b")
			}
		case 3: // Ctrl+C
			return "", true
		default:
			line = append(line, b)
			fmt.Fprintf(w, "%c", b)
		}
	}
}

// runWizardLine is the original plain-text wizard: numbered options,
// typed answers, one line at a time. Used whenever stdin/stdout aren't
// both real terminals. Ctrl+C in cooked mode is a real SIGINT; the
// handler prints "cancelled" and exits 130, matching the TUI path's
// exit code.
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

	a.Theme = askChoice(in, out, "Theme", themeOptions, "default")
	projectTypes := spec.projectTypeOptions()
	a.ProjectType = askChoice(in, out, "Project type", projectTypes, spec.defaultFor(projectTypes, "backend-service"))
	langOpts := spec.languageOptions(a.ProjectType)
	a.Language = askChoice(in, out, "Language", langOpts, spec.defaultFor(langOpts, "go"))
	fwOpts := spec.frameworkOptions(a.ProjectType, a.Language)
	a.Framework = askChoice(in, out, "Framework", fwOpts, spec.defaultFor(fwOpts, "rest-api"))
	if len(spec.Capabilities) > 0 {
		a.Capabilities = askMulti(in, out, "Capabilities (comma-separated numbers or ids, blank for none)", spec.capabilityOptions())
	}

	return a, a.Validate()
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

// askChoice prints numbered options (marking unavailable ones "(coming
// soon)"), and reprompts until an available option is chosen by number
// or id.
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
					fmt.Fprintf(out, "%q is coming soon; please choose an available option.\n", o.Name)
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
