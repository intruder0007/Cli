// Package engine orchestrates one "bootstrap new" run: validate answers,
// resolve the matching template plugin, generate the project, then apply
// each selected capability plugin in order.
package engine

import (
	"fmt"

	"github.com/intruder0007/Cli/core/config"
	"github.com/intruder0007/Cli/core/plugin"
	"github.com/intruder0007/Cli/core/registry"
	sdk "github.com/intruder0007/Cli/sdk/go/sdk"
)

// Summary aggregates the result of a full generate+apply run.
type Summary struct {
	FilesWritten []string
	NextSteps    []string
}

// Engine ties a plugin registry and a plugin host together.
type Engine struct {
	Registry *registry.Registry
	Host     *plugin.Host
}

// New returns an Engine using the given registry and host.
func New(reg *registry.Registry, host *plugin.Host) *Engine {
	return &Engine{Registry: reg, Host: host}
}

func answersMap(a config.Answers) map[string]string {
	return map[string]string{
		"theme":       a.Theme,
		"projectType": a.ProjectType,
		"language":    a.Language,
		"framework":   a.Framework,
	}
}

// Run validates answers, generates the project from the matching template
// plugin, then applies every selected capability plugin in the order the
// user chose them.
func (e *Engine) Run(targetDir string, a config.Answers) (Summary, error) {
	var summary Summary

	if err := a.Validate(); err != nil {
		return summary, err
	}

	tmpl, err := e.Registry.ResolveTemplate(a.ProjectType, a.Language, a.Framework)
	if err != nil {
		return summary, err
	}

	genResp, err := e.Host.Generate(tmpl.EntrypointPath, sdk.GenerateRequest{
		TargetDir:   targetDir,
		ProjectName: a.ProjectName,
		Answers:     answersMap(a),
	})
	if err != nil {
		return summary, fmt.Errorf("generating from template %q: %w", tmpl.Manifest.Name, err)
	}
	summary.FilesWritten = append(summary.FilesWritten, genResp.FilesWritten...)
	summary.NextSteps = append(summary.NextSteps, genResp.NextSteps...)

	for _, capID := range a.Capabilities {
		cap, err := e.Registry.ResolveCapability(capID)
		if err != nil {
			return summary, err
		}
		applyResp, err := e.Host.Apply(cap.EntrypointPath, sdk.ApplyRequest{
			TargetDir:   targetDir,
			ProjectName: a.ProjectName,
			Answers:     answersMap(a),
		})
		if err != nil {
			return summary, fmt.Errorf("applying capability %q: %w", capID, err)
		}
		summary.FilesWritten = append(summary.FilesWritten, applyResp.FilesWritten...)
		summary.NextSteps = append(summary.NextSteps, applyResp.NextSteps...)
	}

	return summary, nil
}
