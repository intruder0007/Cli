// Package registry discovers plugins (templates and capabilities) by
// scanning local directories for plugin.json manifests, and resolves
// wizard answers to the plugin that should handle them. See
// docs/architecture/plugin-protocol.md "Discovery".
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	sdk "github.com/intruder0007/Cli/sdk/go/sdk"
)

// Plugin pairs a discovered manifest with the absolute path to its
// executable entrypoint.
type Plugin struct {
	Manifest       sdk.Manifest
	EntrypointPath string
}

// Registry discovers plugins under a fixed set of local directories.
type Registry struct {
	dirs []string
}

// New returns a Registry that scans the given directories (each
// directory's immediate subdirectories are checked for a plugin.json).
func New(dirs ...string) *Registry {
	return &Registry{dirs: dirs}
}

// Discover scans all configured directories and returns every plugin
// found. A missing directory is skipped, not an error (V1 ships with
// only some of templates/, plugins/ present depending on what's merged).
func (r *Registry) Discover() ([]Plugin, error) {
	var found []Plugin
	for _, dir := range r.dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("registry: reading %s: %w", dir, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			pluginDir := filepath.Join(dir, entry.Name())
			manifestPath := filepath.Join(pluginDir, "plugin.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				continue // not a plugin directory
			}
			var m sdk.Manifest
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, fmt.Errorf("registry: parsing %s: %w", manifestPath, err)
			}
			found = append(found, Plugin{
				Manifest:       m,
				EntrypointPath: filepath.Join(pluginDir, m.Entrypoint),
			})
		}
	}
	return found, nil
}

// ResolveTemplate finds the (V1: single) template plugin matching the
// given project type, language, and framework.
func (r *Registry) ResolveTemplate(projectType, language, framework string) (Plugin, error) {
	plugins, err := r.Discover()
	if err != nil {
		return Plugin{}, err
	}
	for _, p := range plugins {
		if p.Manifest.Kind != "template" {
			continue
		}
		if p.Manifest.ProjectType == projectType && p.Manifest.Language == language && p.Manifest.Framework == framework {
			return p, nil
		}
	}
	return Plugin{}, fmt.Errorf("no template plugin found for project type %q, language %q, framework %q", projectType, language, framework)
}

// ResolveCapability finds a capability plugin by its capabilityId.
func (r *Registry) ResolveCapability(capabilityID string) (Plugin, error) {
	plugins, err := r.Discover()
	if err != nil {
		return Plugin{}, err
	}
	for _, p := range plugins {
		if p.Manifest.Kind == "capability" && p.Manifest.CapabilityID == capabilityID {
			return p, nil
		}
	}
	return Plugin{}, fmt.Errorf("no capability plugin found for id %q", capabilityID)
}
