// Package prompt implements the interactive wizard, the theme registry,
// theme persistence, and a minimal parser for --answers files. It is the
// only place cli talks to a terminal, so the terminal/theme library
// choice stays swappable without touching core. See docs/cli/usage.md
// and ADR-0007.
package prompt

import (
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

var themeOptions = []option{
	{"default", "Default (color + icons)", true},
	{"minimal", "Minimal (plain text, NO_COLOR-friendly)", true},
}

var projectTypes = []option{
	{"backend-service", "Backend Service", true},
	{"web-app", "Web App", false},
	{"cli-tool", "CLI Tool", false},
	{"library", "Library", false},
}

var languages = []option{
	{"go", "Go", true},
	{"node", "Node.js", true},
	{"typescript", "TypeScript", false},
	{"python", "Python", false},
	{"rust", "Rust", false},
}

// frameworks isn't filtered by the selected language in V1 — every
// framework is shown regardless of language, and picking a combination
// with no matching template (e.g. go + http-api) fails with a clear
// registry.TemplateNotFoundError, same as any other "coming soon"
// combination. A per-language framework list is a reasonable future
// improvement, not required for V1's two real templates.
var frameworks = []option{
	{"rest-api", "REST API (net/http)", true},
	{"http-api", "HTTP API (node:http)", true},
	{"grpc", "gRPC", false},
	{"graphql", "GraphQL", false},
}

var capabilityOptions = []option{
	{"git-init", "Initialize Git repository", true},
	{"readme", "Enhance README", true},
	{"github-actions-ci", "GitHub Actions CI (go build + go test)", true},
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
