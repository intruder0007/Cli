package prompt

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"time"

	"github.com/intruder0007/Lumo/core/engine"
	"github.com/intruder0007/Lumo/core/plugin"
	"github.com/intruder0007/Lumo/core/registry"
)

// screens.go — the non-interactive screens and the welcome screen.
// Every screen is composed from the component library (components.go);
// nothing here writes raw ANSI.

// Banner prints the small startup wordmark before the line-fallback
// (piped/non-terminal) wizard. The interactive TUI path renders the
// richer WelcomeScreen instead. Deliberately small — a professional tool
// doesn't need large ASCII art (design-spec §1.2).
func Banner(w io.Writer, t Theme) {
	SetTitle("Lumo — New Project")
	fmt.Fprintln(w, BrandHeader(t))
}

// WelcomeScreen is the interactive landing screen (design-spec S01,
// §7): the brand wordmark, a one-line explanation of what Lumo does, the
// version/runtime identity, and the keyboard hints. It answers "where am
// I" (brand), "what is this" (caption), and "what can I do" (hints) in
// one calm screen.
func WelcomeScreen(w io.Writer, t Theme, version string) {
	o := NewOutput(w)
	o.ClearScreen()
	SetTitle("Lumo — New Project")

	lines := []string{
		"",
		BrandHeader(t),
		"",
		Caption(t, "Scaffold a project in seconds: describe the stack — template, language,"),
		Caption(t, "framework, capabilities — and Lumo generates a tested, ready-to-run project."),
		"",
		t.Dim(fmt.Sprintf("lumo %s · %s · %s/%s", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)),
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

// successProjectRows builds the project-information dashboard rows for
// the success screen (project path, language, framework, capabilities),
// omitting empty ones.
func successProjectRows(t Theme, projectName string, s engine.Summary) []string {
	var rows []string
	rows = append(rows, PanelRow(t, "Name:      ", projectName))
	if s.TargetDir != "" {
		rows = append(rows, PanelRow(t, "Path:      ", s.TargetDir))
	}
	if s.Answers.Language != "" {
		rows = append(rows, PanelRow(t, "Language:  ", languageDisplayName(s.Answers.Language)))
	}
	if s.Answers.Framework != "" {
		rows = append(rows, PanelRow(t, "Framework: ", s.Answers.Framework))
	}
	if len(s.CapabilitiesApplied) > 0 {
		rows = append(rows, PanelRow(t, "Extras:    ", strings.Join(s.CapabilitiesApplied, ", ")))
	}
	return rows
}

// SuccessScreen renders the result of a successful "lumo new" run as a
// dashboard: the success line, a project-information panel, the grouped
// file tree, next steps, and a time note when the run was slow.
func SuccessScreen(w io.Writer, t Theme, projectName string, s engine.Summary, elapsed time.Duration) {
	SetTitle("Lumo — Success")
	fmt.Fprintln(w, t.Success(fmt.Sprintf("Generated %s", projectName)))

	summaryLine := fmt.Sprintf("template: %s · %d file(s)", s.Template, len(s.FilesWritten))
	if len(s.CapabilitiesApplied) > 0 {
		summaryLine += fmt.Sprintf(" · capabilities: %s", strings.Join(s.CapabilitiesApplied, ", "))
	}
	fmt.Fprintln(w, t.Dim("  "+summaryLine))

	if rows := successProjectRows(t, projectName, s); len(rows) > 0 {
		fmt.Fprintln(w)
		for _, l := range StatusPanel(t, "Project", rows) {
			fmt.Fprintln(w, "  "+l)
		}
	}

	// File tree: grouped by producer when the engine returned groups
	// (design-spec §6.1 rule 1), else the flat list.
	fmt.Fprintln(w)
	tree := groupedFileTree(t, s)
	if len(tree) == 0 {
		fmt.Fprintln(w, t.Dim("  no files written"))
	}
	for _, l := range tree {
		fmt.Fprintln(w, "  "+l)
	}

	if len(s.NextSteps) > 0 {
		fmt.Fprintln(w)
		// The first next step is always a "cd into the project"; plugins
		// may already return one themselves, so only prepend when they
		// didn't (design-spec §7: never show the same step twice).
		steps := s.NextSteps
		if !strings.HasPrefix(steps[0], "cd ") {
			steps = append([]string{"cd " + projectName}, steps...)
		}
		for _, l := range NextSteps(t, steps) {
			fmt.Fprintln(w, l)
		}
	}
	if elapsed > 5*time.Second {
		fmt.Fprintln(w)
		fmt.Fprintln(w, t.Dim("completed in "+formatDuration(elapsed)))
	}
}

func groupedFileTree(t Theme, s engine.Summary) []string {
	if len(s.FileGroups) == 0 {
		return FileTree(t, s.FilesWritten)
	}
	var lines []string
	first := true
	for _, g := range s.FileGroups {
		if len(g.Files) == 0 {
			continue
		}
		if !first {
			lines = append(lines, "")
		}
		first = false
		lines = append(lines, t.Dim("  "+g.Label+":"))
		for _, l := range FileTree(t, g.Files) {
			lines = append(lines, "  "+l)
		}
	}
	if len(lines) == 0 {
		return FileTree(t, s.FilesWritten)
	}

	return lines
}

// cancel line printed when the wizard is cancelled (S10).
func printCancelled(w io.Writer, t Theme) {
	fmt.Fprintln(w, t.Info("cancelled"))
}

// ErrorScreen renders a failure (O-04 Failure): what happened (failure
// line), why/next step (hint), and a documentation reference where one
// exists. Receives the error, never a raw stack trace, and sanitizes
// executable paths out of the message (design-spec §8.8: the screen
// never exposes the plugin binary's name or location).
func ErrorScreen(w io.Writer, t Theme, err error) {
	SetTitle("Lumo — Error")
	fmt.Fprintln(w)
	fmt.Fprintln(w, t.Failure(errorMessage(err)))
	if hint := suggestHint(err); hint != "" {
		fmt.Fprintln(w, t.Dim("  "+hint))
	}
	if ref := docsRef(err); ref != "" {
		fmt.Fprintln(w, t.Dim("  reference: "+ref))
	}
	fmt.Fprintln(w)
}

// errorMessage renders err for the failure screen, replacing a plugin
// entrypoint's path with <binary> — the event ("could not start the
// plugin binary") matters, the absolute location on disk doesn't (it
// leaks the filesystem layout / CWD and varies machine to machine).
func errorMessage(err error) string {
	var startErr *plugin.StartError
	if errors.As(err, &startErr) && startErr.EntrypointPath != "" {
		msg := err.Error()
		// The entrypoint path appears both bare ("starting <path>:") and
		// quoted with doubled backslashes from the wrapped exec error
		// (strconv.Quote) — scrub both forms.
		msg = strings.ReplaceAll(msg, startErr.EntrypointPath, "<binary>")
		msg = strings.ReplaceAll(msg, `"`+strings.ReplaceAll(startErr.EntrypointPath, `\`, `\\`)+`"`, "<binary>")
		msg = strings.ReplaceAll(msg, `"<binary>"`, "<binary>")
		return msg
	}
	return err.Error()
}

// suggestHint matches typed errors (errors.As walks the %w chain) and
// falls back to string matching only for errors not yet typed.
func suggestHint(err error) string {
	var notFound *registry.TemplateNotFoundError
	if errors.As(err, &notFound) {
		return "hint: run `lumo plugins list` to see what's discoverable, and check LUMO_PLUGIN_DIRS if templates/ isn't next to the binary."
	}
	var capNotFound *registry.CapabilityNotFoundError
	if errors.As(err, &capNotFound) {
		return "hint: run `lumo plugins list` — the capability id must match a discovered plugin.json's capabilityId exactly."
	}
	var startErr *plugin.StartError
	if errors.As(err, &startErr) {
		return "hint: the plugin binary may be missing or not executable — rebuild it (see docs/plugins/authoring.md / docs/templates/authoring.md)."
	}
	var protoErr *plugin.ProtocolMismatchError
	if errors.As(err, &protoErr) {
		return "hint: the plugin was built against a different protocol version — rebuild it against the current sdk/go."
	}
	var identityErr *plugin.IdentityMismatchError
	if errors.As(err, &identityErr) {
		return "hint: the running plugin binary doesn't match its plugin.json — rebuild it, or check for a stale binary left over from another plugin."
	}
	var timeoutErr *plugin.TimeoutError
	if errors.As(err, &timeoutErr) {
		return "hint: the plugin didn't respond in time — it may be hung; check its logs (stderr) for what it was doing."
	}
	var cycleErr *engine.CapabilityCycleError
	if errors.As(err, &cycleErr) {
		return "hint: the selected capabilities' dependsOn declarations form a cycle — check each capability's plugin.json."
	}

	msg := err.Error()
	if strings.Contains(msg, "project name") {
		return "hint: project names must start with a letter and contain only letters, digits, - or _."
	}
	if strings.Contains(msg, "not empty") || strings.Contains(msg, "is a file") {
		return "hint: pick a different name, or generate into an empty/new directory."
	}
	return ""
}

// docsRef returns a documentation reference for a known failure class,
// or "" when none applies.
func docsRef(err error) string {
	var notFound *registry.TemplateNotFoundError
	var capNotFound *registry.CapabilityNotFoundError
	if errors.As(err, &notFound) || errors.As(err, &capNotFound) {
		return "docs/plugins/authoring.md · docs/cli/usage.md (`lumo doctor`)"
	}
	if isPluginErr(err) {
		return "docs/plugins/authoring.md — plugin binary/protocol"
	}
	var cycleErr *engine.CapabilityCycleError
	if errors.As(err, &cycleErr) {
		return "docs/plugins/authoring.md — dependsOn"
	}
	return ""
}

func isPluginErr(err error) bool {
	var startErr *plugin.StartError
	var protoErr *plugin.ProtocolMismatchError
	var identityErr *plugin.IdentityMismatchError
	var timeoutErr *plugin.TimeoutError
	return errors.As(err, &startErr) || errors.As(err, &protoErr) ||
		errors.As(err, &identityErr) || errors.As(err, &timeoutErr)
}

// HelpText is the top-level `lumo`/`lumo help` output.
const HelpText = `usage: lumo <command> [flags]

Running lumo with no arguments starts the interactive wizard
(the same as 'lumo new').

commands:
  new [project-name]   generate a new project (interactive wizard if no
                        flags/answers given; run 'lumo new -h' for
                        non-interactive flags)
  plugins list          list discovered template and capability plugins
  plugins validate <dir>
                        check a plugin directory before shipping it:
                        manifest validity + binary identity/protocol
                        handshake (run 'lumo plugins validate -h'
                        for details)
  config get theme      print the persisted theme (empty if unset)
  config set theme <name>
                        persist a theme (default|minimal) for future
                        interactive runs
  doctor                 run local health checks (plugin discovery/
                        validity) and report pass/fail with hints
  version                print the CLI version, Go runtime, and platform

Run 'lumo <command> -h' for flags on a specific command.
Pass -verbose (or -v) to 'lumo new' for diagnostic logging on stderr.`
