package registry

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writePlugin(t *testing.T, dir, name, manifestJSON string) {
	t.Helper()
	pluginDir := filepath.Join(dir, name)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifestJSON), 0o644); err != nil {
		t.Fatal(err)
	}
}

const validTemplateJSON = `{
  "protocolVersion": "1", "name": "t", "version": "0.1.0", "kind": "template",
  "projectType": "backend-service", "language": "go", "framework": "rest-api",
  "entrypoint": "./t"
}`

const invalidJSON = `{ this is not valid json`

const missingFieldsJSON = `{"protocolVersion": "1", "name": "broken", "kind": "template", "entrypoint": "./broken"}`

func TestDiscoverSkipsInvalidManifestsWithoutFailingOthers(t *testing.T) {
	dir := t.TempDir()
	writePlugin(t, dir, "good", validTemplateJSON)
	writePlugin(t, dir, "bad-json", invalidJSON)
	writePlugin(t, dir, "bad-fields", missingFieldsJSON)

	r := New(dir)
	plugins, err := r.Discover()
	if err != nil {
		t.Fatalf("Discover should not hard-fail on invalid manifests, got: %v", err)
	}
	if len(plugins) != 1 || plugins[0].Manifest.Name != "t" {
		t.Errorf("Discover() = %v, want exactly the one valid plugin", plugins)
	}
}

func TestDiscoverWithIssuesReportsSkipped(t *testing.T) {
	dir := t.TempDir()
	writePlugin(t, dir, "good", validTemplateJSON)
	writePlugin(t, dir, "bad-json", invalidJSON)
	writePlugin(t, dir, "bad-fields", missingFieldsJSON)

	r := New(dir)
	plugins, issues, err := r.DiscoverWithIssues()
	if err != nil {
		t.Fatalf("DiscoverWithIssues: %v", err)
	}
	if len(plugins) != 1 {
		t.Errorf("got %d valid plugins, want 1", len(plugins))
	}
	if len(issues) != 2 {
		t.Errorf("got %d issues, want 2 (bad-json, bad-fields)", len(issues))
	}
}

func TestResolveTemplateNotFoundIsTyped(t *testing.T) {
	dir := t.TempDir()
	r := New(dir)
	_, err := r.ResolveTemplate("backend-service", "go", "rest-api")
	var notFound *TemplateNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("ResolveTemplate on empty registry: got err=%v (%T), want *TemplateNotFoundError", err, err)
	}
}

func TestResolveCapabilityNotFoundIsTyped(t *testing.T) {
	dir := t.TempDir()
	r := New(dir)
	_, err := r.ResolveCapability("nope")
	var notFound *CapabilityNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("ResolveCapability on empty registry: got err=%v (%T), want *CapabilityNotFoundError", err, err)
	}
}

func TestResolveTemplateFindsMatch(t *testing.T) {
	dir := t.TempDir()
	writePlugin(t, dir, "t", validTemplateJSON)
	r := New(dir)
	p, err := r.ResolveTemplate("backend-service", "go", "rest-api")
	if err != nil {
		t.Fatalf("ResolveTemplate: %v", err)
	}
	if p.Manifest.Name != "t" {
		t.Errorf("got %q, want %q", p.Manifest.Name, "t")
	}
}

func TestDiscoverMissingDirIsNotAnError(t *testing.T) {
	r := New(filepath.Join(t.TempDir(), "does-not-exist"))
	plugins, err := r.Discover()
	if err != nil {
		t.Errorf("missing directory should not be an error, got: %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("got %d plugins from a missing directory, want 0", len(plugins))
	}
}
