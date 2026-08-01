// Package embedded is a last-resort fallback for plugin discovery: the
// V1 template and capability binaries, staged into this package's
// assets/ directory at build time (see the Makefile and
// .github/workflows/release.yml) and compiled into the cli binary
// itself via go:embed. It is used only when no sibling plugin
// directories can be found at all — see cli/main.go's pluginDirs().
// This deliberately lives in cli, not core: core's own principle is
// that it never hardcodes what it can generate, and baking the
// specific V1 plugin set in would violate that in spirit even though
// go:embed doesn't create a Go import. Only cli, the distribution-facing
// binary, knows which plugins V1 ships.
package embedded

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:assets
var assetsFS embed.FS

const assetsRoot = "assets"

// Available reports whether any plugin assets were actually staged into
// this binary at build time. A plain `go build ./cli`, run without the
// Makefile's staging step, embeds only the checked-in assets/.gitkeep
// placeholder — Available reports false for that build, and ExtractTo
// becomes a harmless no-op.
func Available() bool {
	return dirHasSubdirs(assetsFS, assetsRoot)
}

func dirHasSubdirs(fsys fs.FS, root string) bool {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			return true
		}
	}
	return false
}

// ExtractTo writes the embedded assets to dir, creating it as needed.
// Idempotent: if dir already exists and is non-empty, it's left alone —
// callers are expected to pass a version-scoped directory, so a
// non-empty dir means a previous run already extracted this exact
// version.
func ExtractTo(dir string) error {
	return extractTo(assetsFS, assetsRoot, dir)
}

// extractTo does the real work against an injected fs.FS, so it's
// testable against a synthetic filesystem (fstest.MapFS) without
// depending on real staged assets being present at test time.
func extractTo(fsys fs.FS, root, dir string) error {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return nil
	}
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return extractEntry(fsys, root, dir, path, d)
	})
}

// extractEntry handles one WalkDir-visited path: skips the root itself
// and the .gitkeep placeholder, recreates directories, and writes files
// (marking anything other than plugin.json executable, since the only
// files this package ever stages are plugin.json manifests and plugin
// entrypoint binaries).
func extractEntry(fsys fs.FS, root, dir, path string, d fs.DirEntry) error {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	if rel == "." || rel == ".gitkeep" {
		return nil
	}
	target := filepath.Join(dir, rel)
	if d.IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o755) // plugin entrypoint binaries must be executable
	if filepath.Base(target) == "plugin.json" {
		mode = 0o644
	}
	return os.WriteFile(target, data, mode)
}
