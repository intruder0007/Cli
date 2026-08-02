// Package engine orchestrates one "bootstrap new" run: validate answers,
// resolve the matching template plugin and every selected capability
// plugin up front (fail-fast, before anything runs), order capabilities
// respecting any declared dependencies, generate the project, then apply
// each capability in that order.
package engine

import (
	"fmt"

	"github.com/intruder0007/Cli/core/config"
	"github.com/intruder0007/Cli/core/diag"
	"github.com/intruder0007/Cli/core/registry"
	sdk "github.com/intruder0007/Cli/sdk/go/sdk"
)

// Summary aggregates the result of a full generate+apply run.
type Summary struct {
	FilesWritten        []string
	NextSteps           []string
	Template            string
	CapabilitiesApplied []string
}

// Resolver finds plugins by wizard answer / capability id. Satisfied by
// *registry.Registry; an interface here so Engine.Run's fail-fast and
// dependency-ordering logic can be unit-tested with a fake, no plugin
// discovery or subprocess needed.
type Resolver interface {
	ResolveTemplate(projectType, language, framework string) (registry.Plugin, error)
	ResolveCapability(capabilityID string) (registry.Plugin, error)
}

// Runner invokes a resolved plugin. Satisfied by *plugin.Host.
type Runner interface {
	Generate(entrypointPath, expectedName, expectedProtocolVersion string, req sdk.GenerateRequest) (sdk.GenerateResponse, error)
	Apply(entrypointPath, expectedName, expectedProtocolVersion string, req sdk.ApplyRequest) (sdk.ApplyResponse, error)
}

// Engine ties a plugin registry and a plugin host together.
type Engine struct {
	Registry Resolver
	Host     Runner
	// Logger receives diagnostic lines about resolution/ordering
	// decisions. Nil means silent.
	Logger diag.Logger
}

// New returns an Engine using the given registry and host.
func New(reg Resolver, host Runner) *Engine {
	return &Engine{Registry: reg, Host: host}
}

func (e *Engine) logger() diag.Logger {
	if e.Logger != nil {
		return e.Logger
	}
	return diag.NoopLogger{}
}

// CapabilityCycleError means the selected capabilities' dependsOn
// declarations form a cycle that can't be ordered.
type CapabilityCycleError struct {
	CapabilityIDs []string
}

func (e *CapabilityCycleError) Error() string {
	return fmt.Sprintf("capabilities have a dependency cycle among: %v", e.CapabilityIDs)
}

func answersMap(a config.Answers) map[string]string {
	return map[string]string{
		"theme":       a.Theme,
		"projectType": a.ProjectType,
		"language":    a.Language,
		"framework":   a.Framework,
	}
}

// Run validates answers, resolves the template and every selected
// capability up front (so a bad capability id fails before anything has
// written a file), orders capabilities by any declared dependencies,
// generates the project, then applies each capability in that order.
func (e *Engine) Run(targetDir string, a config.Answers) (Summary, error) {
	var summary Summary
	log := e.logger()

	if err := a.Validate(); err != nil {
		return summary, err
	}

	tmpl, err := e.Registry.ResolveTemplate(a.ProjectType, a.Language, a.Framework)
	if err != nil {
		return summary, err
	}
	log.Logf("resolved template %q for %s/%s/%s", tmpl.Manifest.Name, a.ProjectType, a.Language, a.Framework)

	ordered, err := e.resolveOrderedCapabilities(a.Capabilities)
	if err != nil {
		return summary, err
	}
	if len(ordered) > 0 {
		log.Logf("resolved %d capabilities, execution order: %v", len(ordered), capabilityIDs(ordered))
	}

	summary.Template = tmpl.Manifest.Name

	genResp, err := e.Host.Generate(tmpl.EntrypointPath, tmpl.Manifest.Name, sdk.ProtocolVersion, sdk.GenerateRequest{
		TargetDir:   targetDir,
		ProjectName: a.ProjectName,
		Answers:     answersMap(a),
	})
	if err != nil {
		return summary, fmt.Errorf("generating from template %q: %w", tmpl.Manifest.Name, err)
	}
	summary.FilesWritten = append(summary.FilesWritten, genResp.FilesWritten...)
	summary.NextSteps = append(summary.NextSteps, genResp.NextSteps...)

	for _, c := range ordered {
		applyResp, err := e.Host.Apply(c.EntrypointPath, c.Manifest.Name, sdk.ProtocolVersion, sdk.ApplyRequest{
			TargetDir:   targetDir,
			ProjectName: a.ProjectName,
			Answers:     answersMap(a),
		})
		if err != nil {
			return summary, fmt.Errorf("applying capability %q: %w", c.Manifest.CapabilityID, err)
		}
		summary.FilesWritten = append(summary.FilesWritten, applyResp.FilesWritten...)
		summary.NextSteps = append(summary.NextSteps, applyResp.NextSteps...)
		summary.CapabilitiesApplied = append(summary.CapabilitiesApplied, c.Manifest.CapabilityID)
	}

	log.Logf("run complete: %d files written, %d capabilities applied", len(summary.FilesWritten), len(summary.CapabilitiesApplied))
	return summary, nil
}

// resolveOrderedCapabilities resolves every selected capability id
// (fail-fast: no subprocess spawned yet, so a bad id is caught before
// anything runs) and returns them ordered by any declared dependencies.
// Duplicate ids in the selection are collapsed (first occurrence wins),
// so `--capabilities git-init,git-init` runs the capability once.
func (e *Engine) resolveOrderedCapabilities(capabilityIDs []string) ([]registry.Plugin, error) {
	seen := make(map[string]bool, len(capabilityIDs))
	caps := make([]registry.Plugin, 0, len(capabilityIDs))
	for _, capID := range capabilityIDs {
		if seen[capID] {
			continue
		}
		seen[capID] = true
		c, err := e.Registry.ResolveCapability(capID)
		if err != nil {
			return nil, err
		}
		caps = append(caps, c)
	}
	return sortByDependencies(caps)
}

// sortByDependencies orders caps so that every plugin's DependsOn
// entries (among capability ids present in caps — an entry naming a
// capability the user didn't select is ignored) come before it. Ties
// (no ordering constraint between two plugins) are broken by caps'
// original order, so a run with no dependsOn declarations at all — true
// for every V1 capability — produces exactly the user's selection order,
// unchanged. Kahn's algorithm, scanning in original order each pass for
// full determinism.
func capabilityIDs(caps []registry.Plugin) []string {
	out := make([]string, len(caps))
	for i, c := range caps {
		out[i] = c.Manifest.CapabilityID
	}
	return out
}

func sortByDependencies(caps []registry.Plugin) ([]registry.Plugin, error) {
	index := make(map[string]int, len(caps))
	for i, c := range caps {
		index[c.Manifest.CapabilityID] = i
	}

	inDegree := make([]int, len(caps))
	dependents := make([][]int, len(caps))
	for i, c := range caps {
		for _, dep := range c.Manifest.DependsOn {
			j, ok := index[dep]
			if !ok {
				continue // dependency not selected by the user: ignored
			}
			dependents[j] = append(dependents[j], i)
			inDegree[i]++
		}
	}

	used := make([]bool, len(caps))
	ordered := make([]registry.Plugin, 0, len(caps))
	for len(ordered) < len(caps) {
		progressed := false
		for i := range caps {
			if used[i] || inDegree[i] > 0 {
				continue
			}
			used[i] = true
			ordered = append(ordered, caps[i])
			progressed = true
			for _, j := range dependents[i] {
				inDegree[j]--
			}
		}
		if !progressed {
			var remaining []string
			for i, u := range used {
				if !u {
					remaining = append(remaining, caps[i].Manifest.CapabilityID)
				}
			}
			return nil, &CapabilityCycleError{CapabilityIDs: remaining}
		}
	}
	return ordered, nil
}
