// Package prompt implements the interactive wizard, the theme registry,
// theme persistence, and a minimal parser for --answers files. It is the
// only place cli talks to a terminal, so the terminal/theme library
// choice stays swappable without touching core. See docs/cli/usage.md
// and ADR-0007.
package prompt

import (
	"fmt"
	"os"
	"strings"

	"github.com/intruder0007/Lumo/core/config"
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

// TemplateSpec is one discovered template plugin, reduced to the fields
// the wizard renders menus from: its exact project type + language +
// framework triple (the registry's ResolveTemplate match key) and the
// display name shown for it.
type TemplateSpec struct {
	ProjectType string
	Language    string
	Framework   string
	DisplayName string
}

// CapabilitySpec is one discovered capability plugin.
type CapabilitySpec struct {
	ID          string
	DisplayName string
}

// WizardSpec is the discoverable surface of the installed plugin set —
// everything the wizard can offer. It is plain data: main.go builds it
// from core/registry.Discover(), keeping prompt decoupled from the
// registry (ADR-0007). An empty Templates list means nothing can be
// generated, and RunWizard fails fast rather than asking dead-end
// questions about options that can't resolve.
type WizardSpec struct {
	Templates    []TemplateSpec
	Capabilities []CapabilitySpec
}

// projectTypeOptions returns the distinct project types among the
// discovered templates, in discovery order. Every option is available —
// the wizard only offers what's actually installed.
func (s WizardSpec) projectTypeOptions() []option {
	seen := make(map[string]bool)
	var out []option
	for _, t := range s.Templates {
		if seen[t.ProjectType] {
			continue
		}
		seen[t.ProjectType] = true
		out = append(out, option{ID: t.ProjectType, Name: humanizeID(t.ProjectType), Available: true})
	}
	return out
}

// languageOptions returns the languages offered for one project type —
// a template exists for every returned combination.
func (s WizardSpec) languageOptions(projectType string) []option {
	seen := make(map[string]bool)
	var out []option
	for _, t := range s.Templates {
		if t.ProjectType != projectType || seen[t.Language] {
			continue
		}
		seen[t.Language] = true
		out = append(out, option{ID: t.Language, Name: languageDisplayName(t.Language), Available: true})
	}
	return out
}

// frameworkOptions returns the frameworks offered for one (project
// type, language) pair — fixing V1's limitation where every framework
// was shown regardless of language, and a dead-end combination failed
// only at the end with a registry.TemplateNotFoundError.
func (s WizardSpec) frameworkOptions(projectType, language string) []option {
	var out []option
	for _, t := range s.Templates {
		if t.ProjectType != projectType || t.Language != language {
			continue
		}
		out = append(out, option{ID: t.Framework, Name: t.DisplayName, Available: true})
	}
	return out
}

func (s WizardSpec) capabilityOptions() []option {
	var out []option
	for _, c := range s.Capabilities {
		out = append(out, option{ID: c.ID, Name: c.DisplayName, Available: true})
	}
	return out
}

// defaultFor returns preferred if it names one of opts, else the first
// option — so a wizard default always names a real option even when
// the preferred id isn't installed (the menus' own cursor fallback uses
// the same rule; this is the non-interactive/line-wizard twin of it).
func (s WizardSpec) defaultFor(opts []option, preferred string) string {
	for _, o := range opts {
		if o.ID == preferred {
			return preferred
		}
	}
	if len(opts) > 0 {
		return opts[0].ID
	}
	return ""
}

// languageNames is the CLI's vocabulary of human names for well-known
// language ids, so menus read "Node.js" instead of "node". Unknown ids
// fall back to humanizeID — languages are a small, stable vocabulary,
// unlike project types and frameworks, which come from templates.
var languageNames = map[string]string{
	"go": "Go", "node": "Node.js", "typescript": "TypeScript",
	"python": "Python", "rust": "Rust", "java": "Java", "csharp": "C#",
	"ruby": "Ruby", "php": "PHP", "kotlin": "Kotlin", "swift": "Swift",
	"dart": "Dart",
}

func languageDisplayName(id string) string {
	if n, ok := languageNames[id]; ok {
		return n
	}
	return humanizeID(id)
}

// humanizeID turns an option id into a menu label: "backend-service"
// becomes "Backend Service".
func humanizeID(id string) string {
	words := strings.Split(id, "-")
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
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
	// A file that sets no fields (or only unknown ones) is almost always
	// a path typo or a format mistake — validate the answer shape now, so
	// the user learns about it before any codegen runs. The project-name
	// pattern is intentionally deferred: projectName may hold a target
	// path (e.g. "./a/b/app") that cmdNew resolves to a name + directory
	// before validating the name (see config.Answers.ValidateShape).
	if err := a.ValidateShape(); err != nil {
		return a, fmt.Errorf("validating answers file %s: %w", path, err)
	}
	return a, nil
}
