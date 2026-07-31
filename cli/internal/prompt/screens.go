package prompt

import (
	"fmt"
	"io"
	"strings"
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

func suggestFix(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no template plugin found"):
		return "hint: run `bootstrap plugins list` to see what's discoverable, and check CLI_PLUGIN_DIRS if templates/ isn't next to the binary."
	case strings.Contains(msg, "no capability plugin found"):
		return "hint: run `bootstrap plugins list` — the capability id must match a discovered plugin.json's capabilityId exactly."
	case strings.Contains(msg, "starting"):
		return "hint: the plugin binary may be missing or not executable — rebuild it (see docs/plugins/authoring.md / docs/templates/authoring.md)."
	case strings.Contains(msg, "project name"):
		return "hint: project names must start with a letter and contain only letters, digits, - or _."
	default:
		return ""
	}
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
