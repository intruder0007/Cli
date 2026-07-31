// Package config defines the answers collected by the wizard (or passed
// non-interactively) and validates their shape before the engine resolves
// them against the plugin registry.
package config

import (
	"fmt"
	"regexp"
)

// Answers is the full set of wizard answers for one "bootstrap new" run.
type Answers struct {
	ProjectName  string
	Theme        string   // "default" | "minimal"
	ProjectType  string   // e.g. "backend-service"
	Language     string   // e.g. "go"
	Framework    string   // e.g. "rest-api"
	Capabilities []string // e.g. ["git-init", "readme", "github-actions-ci"]
}

var validThemes = map[string]bool{"default": true, "minimal": true}

var projectNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// Validate checks the shape of Answers (non-empty required fields, a
// filesystem-safe project name, a recognized theme). It does not check
// whether a template/capability actually exists for these answers — that
// is the registry's job.
func (a Answers) Validate() error {
	if a.ProjectName == "" {
		return fmt.Errorf("project name is required")
	}
	if !projectNamePattern.MatchString(a.ProjectName) {
		return fmt.Errorf("project name %q must start with a letter and contain only letters, digits, - or _", a.ProjectName)
	}
	if a.Theme == "" {
		a.Theme = "default"
	}
	if !validThemes[a.Theme] {
		return fmt.Errorf("unknown theme %q (want \"default\" or \"minimal\")", a.Theme)
	}
	if a.ProjectType == "" {
		return fmt.Errorf("project type is required")
	}
	if a.Language == "" {
		return fmt.Errorf("language is required")
	}
	if a.Framework == "" {
		return fmt.Errorf("framework is required")
	}
	return nil
}
