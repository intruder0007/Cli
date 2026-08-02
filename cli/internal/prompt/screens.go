package prompt

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/intruder0007/Lumo/core/engine"
	"github.com/intruder0007/Lumo/core/plugin"
	"github.com/intruder0007/Lumo/core/registry"
)

// Banner prints a small startup wordmark before the interactive wizard
// starts. Deliberately small — a professional tool doesn't need large
// ASCII art, and it must still read cleanly in the minimal theme.
func Banner(w io.Writer, t Theme) {
	if t.UseIcons {
		fmt.Fprintln(w, t.Primary("Lumo")+t.Dim(" — a new project, ready in seconds"))
	} else {
		fmt.Fprintln(w, "Lumo — a new project, ready in seconds")
	}
	fmt.Fprintln(w)
}

// SuccessScreen renders the result of a successful "lumo new" run:
// a project summary (template, capabilities applied, file count), the
// files written, and what to do next.
func SuccessScreen(w io.Writer, t Theme, projectName string, s engine.Summary) {
	fmt.Fprintln(w, t.Success(fmt.Sprintf("Generated %s", projectName)))

	summaryLine := fmt.Sprintf("template: %s · %d file(s)", s.Template, len(s.FilesWritten))
	if len(s.CapabilitiesApplied) > 0 {
		summaryLine += fmt.Sprintf(" · capabilities: %s", strings.Join(s.CapabilitiesApplied, ", "))
	}
	fmt.Fprintln(w, t.Dim("  "+summaryLine))

	if len(s.FilesWritten) > 0 {
		fmt.Fprintln(w)
		for i, f := range s.FilesWritten {
			glyph := "+"
			if t.UseIcons {
				if i == len(s.FilesWritten)-1 {
					glyph = "└─"
				} else {
					glyph = "├─"
				}
			}
			fmt.Fprintln(w, t.Info("  "+glyph+" "+f))
		}
	}
	if len(s.NextSteps) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, t.Header("Next steps:"))
		arrow := "-"
		if t.UseIcons {
			arrow = "→"
		}
		for _, step := range s.NextSteps {
			fmt.Fprintln(w, "  "+t.Accent(arrow)+" "+step)
		}
	}
}

// ErrorScreen renders a failure, with a recovery suggestion when the
// error matches a known shape. This only improves *presentation* of
// whatever error already surfaced — a structured error taxonomy across
// the plugin protocol itself is separate, larger work (see the Phase 3
// roadmap notes in docs/architecture/roadmap.md), not attempted here.
func ErrorScreen(w io.Writer, t Theme, err error) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, t.Failure(err.Error()))
	if hint := suggestFix(err); hint != "" {
		fmt.Fprintln(w, t.Dim("  "+hint))
	}
	fmt.Fprintln(w)
}

// suggestFix matches on typed errors first (errors.As walks the %w
// chain, so this still works through engine.Run's "generating from
// template %q: %w" / "applying capability %q: %w" wrapping), falling
// back to string matching only for errors not yet typed (e.g.
// core/config's plain validation errors).
func suggestFix(err error) string {
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
	return ""
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
