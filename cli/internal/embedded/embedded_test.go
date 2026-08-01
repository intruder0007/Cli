package embedded

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"
)

func fakeAssets() fstest.MapFS {
	return fstest.MapFS{
		"assets/.gitkeep":                              {Data: nil},
		"assets/templates/fake-template/plugin.json":   {Data: []byte(`{"name":"fake-template"}`)},
		"assets/templates/fake-template/fake-template": {Data: []byte("binary")},
		"assets/plugins/builtin/fake-cap/plugin.json":  {Data: []byte(`{"name":"fake-cap"}`)},
		"assets/plugins/builtin/fake-cap/fake-cap":     {Data: []byte("binary")},
	}
}

func TestDirHasSubdirsTrueWhenStaged(t *testing.T) {
	if !dirHasSubdirs(fakeAssets(), assetsRoot) {
		t.Fatal("expected dirHasSubdirs to report true for a filesystem with staged plugin directories")
	}
}

func TestDirHasSubdirsFalseWhenOnlyPlaceholder(t *testing.T) {
	onlyPlaceholder := fstest.MapFS{"assets/.gitkeep": {Data: nil}}
	if dirHasSubdirs(onlyPlaceholder, assetsRoot) {
		t.Fatal("expected dirHasSubdirs to report false when only .gitkeep is present")
	}
}

func TestExtractToWritesExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	if err := extractTo(fakeAssets(), assetsRoot, dir); err != nil {
		t.Fatalf("extractTo: %v", err)
	}

	manifest := filepath.Join(dir, "templates", "fake-template", "plugin.json")
	if _, err := os.Stat(manifest); err != nil {
		t.Fatalf("expected %s to exist: %v", manifest, err)
	}
	bin := filepath.Join(dir, "plugins", "builtin", "fake-cap", "fake-cap")
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("expected %s to exist: %v", bin, err)
	}
	// Windows has no Unix-style execute permission bit — executability
	// there comes from the .exe extension, not file mode — so the
	// requested-executable check only means anything on Unix.
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		t.Fatalf("expected %s to be executable, got mode %v", bin, info.Mode())
	}

	if _, err := os.Stat(filepath.Join(dir, ".gitkeep")); err == nil {
		t.Fatal(".gitkeep should not have been extracted")
	}
}

func TestExtractToIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := extractTo(fakeAssets(), assetsRoot, dir); err != nil {
		t.Fatalf("first extractTo: %v", err)
	}

	sentinel := filepath.Join(dir, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("writing sentinel: %v", err)
	}

	if err := extractTo(fakeAssets(), assetsRoot, dir); err != nil {
		t.Fatalf("second extractTo: %v", err)
	}

	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("expected second extractTo to be a no-op on a non-empty dir, sentinel gone: %v", err)
	}
}
