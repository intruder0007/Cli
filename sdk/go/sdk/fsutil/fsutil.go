// Package fsutil is a small file-writing helper shared by Lumo's own
// template plugins (go-rest-api, node-rest-api, and future language
// templates). It is NOT part of sdk/go/sdk's Stable public API
// (ADR-0013) — sdk/go/sdk is the wire-protocol/plugin-lifecycle
// surface every plugin author depends on, while fsutil is internal
// plumbing for templates that ship with Lumo itself. It may change
// without a deprecation cycle.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// WriteFiles writes files (relative path → content) under targetDir,
// creating parent directories as needed, and returns the relative
// paths written in sorted order — deterministic regardless of map
// iteration order, since callers' FilesWritten is compared against
// other plugins' output (e.g. by the engine's file-tree grouping).
func WriteFiles(targetDir string, files map[string]string) ([]string, error) {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	written := make([]string, 0, len(keys))
	for _, rel := range keys {
		full := filepath.Join(targetDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return written, fmt.Errorf("creating directory for %s: %w", rel, err)
		}
		if err := os.WriteFile(full, []byte(files[rel]), 0o644); err != nil {
			return written, fmt.Errorf("writing %s: %w", rel, err)
		}
		written = append(written, rel)
	}
	return written, nil
}
