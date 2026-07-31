package prompt

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/intruder0007/Cli/core/engine"
	"github.com/intruder0007/Cli/core/plugin"
	"github.com/intruder0007/Cli/core/registry"
)

// Banner prints a small startup wordmark before the interactive wizard
// starts. Deliberately small — a professional tool doesn't need large
// ASCII art, and it must still read cleanly in the minimal theme.
func Banner(w io.Writer, t Theme) {
	if t.UseIcons {
		fmt.Fprintln(w, t.color("1;36", "Cli")+t.Dim("  — bootstrap a new project"))
	} else {
		fmt.Fprintln(w, "Cli — bootstrap a new project")
	}
}

// SuccessScreen renders the result of a successful "bootstrap new" run:
// what was generated, and what to do next.
func SuccessScreen(w io.Writer, t Theme, projectName string, filesWritten, nextSteps []string) {
	fmt.Fprintln(w, t.Success(fmt.Sprintf("Generated %s", projectName)))
	if len(filesWritten) > 0 {
		fmt.Fprintln(w)
		for _, f := range filesWritten {
			fmt.Fprintln(w, t.Info("  + "+f))
		}
	}
	if len(nextSteps) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, t.Header("Next steps:"))
		for _, s := range nextSteps {
			fmt.Fprintln(w, "  "+s)
		}
	}
}

// ErrorScreen renders a failure, with a recovery suggestion when the
// error matches a known shape. This only improves *presentation* of
// whatever error already surfaced — a structured error taxonomy across
// the plugin protocol itself is separate, larger work (see the Phase 3
// roadmap notes in docs/architecture/roadmap.md), not attempted here.
func ErrorScreen(w io.Writer, t Theme, err error) {
	fmt.Fprintln(w, t.Failure(err.Error()))
	if hint := suggestFix(err); hint != "" {
		fmt.Fprintln(w, t.Dim("  "+hint))
	}
}

// suggestFix matches on typed errors first (errors.As walks the %w
// chain, so this still works through engine.Run's "generating from
// template %q: %w" / "applying capability %q: %w" wrapping), falling
// back to string matching only for errors not yet typed (e.g.
// core/config's plain validation errors).
func suggestFix(err error) string {
	var notFound *registry.TemplateNotFoundError
	if errors.As(err, &notFound) {
		return "hint: run `bootstrap plugins list` to see what's discoverable, and check CLI_PLUGIN_DIRS if templates/ isn't next to the binary."
	}
	var capNotFound *registry.CapabilityNotFoundError
	if errors.As(err, &capNotFound) {
		return "hint: run `bootstrap plugins list` — the capability id must match a discovered plugin.json's capabilityId exactly."
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
	return ""
}

// HelpText is the top-level `bootstrap`/`bootstrap help` output.
const HelpText = `usage: bootstrap <command> [flags]

commands:
  new [project-name]   generate a new project (interactive wizard if no
                        flags/answers given; run 'bootstrap new -h' for
                        non-interactive flags)
  plugins list          list discovered template and capability plugins
  config get theme      print the persisted theme (empty if unset)
  config set theme <name>
                        persist a theme (default|minimal) for future
                        interactive runs
  version                print the CLI version

Run 'bootstrap <command> -h' for flags on a specific command.`
