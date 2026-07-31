// Package prompt implements the interactive wizard, the two CLI themes
// (default/minimal), and a minimal parser for --answers files. It is the
// only place cli talks to a terminal, so the theme/prompt library choice
// stays swappable without touching core. See docs/cli/usage.md.
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intruder0007/Cli/core/config"
)

// option is one selectable (or "coming soon") choice in a wizard step.
type option struct {
	ID        string
	Name      string
	Available bool
}

var projectTypes = []option{
	{"backend-service", "Backend Service", true},
	{"web-app", "Web App", false},
	{"cli-tool", "CLI Tool", false},
	{"library", "Library", false},
}

var languages = []option{
	{"go", "Go", true},
	{"typescript", "TypeScript", false},
	{"python", "Python", false},
	{"rust", "Rust", false},
}

var frameworks = []option{
	{"rest-api", "REST API (net/http)", true},
	{"grpc", "gRPC", false},
	{"graphql", "GraphQL", false},
}

var capabilities = []option{
	{"git-init", "Initialize Git repository", true},
	{"readme", "Enhance README", true},
	{"github-actions-ci", "GitHub Actions CI (go build + go test)", true},
}

// Renderer controls how the CLI presents output — the "theme" wizard
// step. minimal never uses color/icons (screen-reader and NO_COLOR
// friendly); default uses color unless explicitly suppressed.
type Renderer struct {
	UseColor bool
	UseIcons bool
}

// NewRenderer builds a Renderer for the given theme ("default" or
// "minimal"). noColor forces color off regardless of theme.
func NewRenderer(theme string, noColor bool) Renderer {
	if theme == "minimal" {
		return Renderer{UseColor: false, UseIcons: false}
	}
	return Renderer{UseColor: !noColor, UseIcons: true}
}

func (r Renderer) color(code, s string) string {
	if !r.UseColor {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

// Success formats a success line. Every state has a text label — color
// and icons are additive, never the only signal.
func (r Renderer) Success(msg string) string {
	label := "[OK] "
	if r.UseIcons {
		label = "✔ "
	}
	return r.color("32", label) + msg
}

// Failure formats a failure line.
func (r Renderer) Failure(msg string) string {
	label := "[FAIL] "
	if r.UseIcons {
		label = "✗ "
	}
	return r.color("31", label) + msg
}

// Info formats an informational line (e.g. a generated file).
func (r Renderer) Info(msg string) string {
	return msg
}

// Header formats a section header.
func (r Renderer) Header(msg string) string {
	return r.color("1;36", msg)
}

// RunWizard walks through project name, theme, project type, language,
// framework, and capabilities, in that order, reading from in and
// writing prompts to out.
func RunWizard(in *bufio.Reader, out io.Writer) (config.Answers, error) {
	var a config.Answers

	a.ProjectName = ask(in, out, "Project name", "")
	if a.ProjectName == "" {
		return a, fmt.Errorf("project name is required")
	}

	a.Theme = askChoice(in, out, "Theme", []option{
		{"default", "Default (color + icons)", true},
		{"minimal", "Minimal (plain text, NO_COLOR-friendly)", true},
	}, "default")

	a.ProjectType = askChoice(in, out, "Project type", projectTypes, "backend-service")
	a.Language = askChoice(in, out, "Language", languages, "go")
	a.Framework = askChoice(in, out, "Framework", frameworks, "rest-api")
	a.Capabilities = askMulti(in, out, "Capabilities (comma-separated numbers or ids, blank for none)", capabilities)

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
			fmt.Fprintf(out, "  %d) %s\n", i+1, o.ID)
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
				if tok == o.ID || tok == fmt.Sprintf("%d", i+1) {
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

// ParseAnswersFile parses a restricted flat "key: value" answers file
// (see docs/cli/usage.md for the format). It intentionally implements
// only the small subset needed here rather than depending on a YAML
// library, keeping the CLI dependency-free.
func ParseAnswersFile(path string) (config.Answers, error) {
	var a config.Answers
	data, err := os.ReadFile(path)
	if err != nil {
		return a, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, `"'`)

		switch key {
		case "projectName":
			a.ProjectName = val
		case "theme":
			a.Theme = val
		case "projectType":
			a.ProjectType = val
		case "language":
			a.Language = val
		case "framework":
			a.Framework = val
		case "capabilities":
			val = strings.TrimPrefix(val, "[")
			val = strings.TrimSuffix(val, "]")
			for _, c := range strings.Split(val, ",") {
				c = strings.TrimSpace(c)
				if c != "" {
					a.Capabilities = append(a.Capabilities, c)
				}
			}
		}
	}
	return a, nil
}
