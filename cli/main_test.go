package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasAnyPluginFalseForEmptyOrMissingDirs(t *testing.T) {
	empty := t.TempDir()
	missing := filepath.Join(empty, "does-not-exist")
	if hasAnyPlugin([]string{empty, missing}) {
		t.Fatal("expected hasAnyPlugin to be false for an empty dir and a missing dir")
	}
}

func TestHasAnyPluginFalseForSubdirWithoutManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "not-a-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if hasAnyPlugin([]string{dir}) {
		t.Fatal("expected hasAnyPlugin to be false when no subdirectory has a plugin.json")
	}
}

func TestHasAnyPluginTrueWhenManifestPresent(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "some-plugin")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasAnyPlugin([]string{dir}) {
		t.Fatal("expected hasAnyPlugin to be true when a subdirectory has a plugin.json")
	}
}

// TestPluginDirsOverrideSkipsFallback verifies that an explicit
// LUMO_PLUGIN_DIRS always wins, even pointing at an empty directory —
// the embedded fallback must never silently override deliberate intent.
func TestPluginDirsOverrideSkipsFallback(t *testing.T) {
	empty := t.TempDir()
	t.Setenv("LUMO_PLUGIN_DIRS", empty)

	dirs := pluginDirs()
	if len(dirs) != 1 || dirs[0] != empty {
		t.Fatalf("expected pluginDirs() to return exactly the override %q unchanged, got %v", empty, dirs)
	}
}

func TestEmbeddedCacheDirIsVersionScoped(t *testing.T) {
	dir, err := embeddedCacheDir()
	if err != nil {
		t.Fatalf("embeddedCacheDir: %v", err)
	}
	if filepath.Base(dir) != version {
		t.Fatalf("expected cache dir %q to end with the current version %q", dir, version)
	}
	if filepath.Base(filepath.Dir(dir)) != "lumo" {
		t.Fatalf("expected cache dir %q to be under a 'lumo' directory", dir)
	}
}
