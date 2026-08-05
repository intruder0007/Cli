package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFilesWritesAndCreatesDirs(t *testing.T) {
	dir := t.TempDir()

	written, err := WriteFiles(dir, map[string]string{
		"go.mod":                 "module x\n",
		"internal/http/route.go": "package http\n",
		".gitignore":             "/bin/\n",
	})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	want := []string{".gitignore", "go.mod", "internal/http/route.go"}
	if len(written) != len(want) {
		t.Fatalf("WriteFiles returned %v, want %v", written, want)
	}
	for i, rel := range want {
		if written[i] != rel {
			t.Errorf("WriteFiles[%d] = %q, want %q (sorted order)", i, written[i], rel)
		}
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("reading back %s: %v", rel, err)
		}
		_ = data
	}
}

func TestWriteFilesDeterministicOrderRegardlessOfMapIteration(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{"c": "3", "a": "1", "b": "2"}

	for i := 0; i < 5; i++ {
		written, err := WriteFiles(dir, files)
		if err != nil {
			t.Fatalf("WriteFiles: %v", err)
		}
		want := []string{"a", "b", "c"}
		for j, rel := range want {
			if written[j] != rel {
				t.Fatalf("run %d: WriteFiles order = %v, want %v", i, written, want)
			}
		}
	}
}

func TestWriteFilesEmptyMap(t *testing.T) {
	written, err := WriteFiles(t.TempDir(), map[string]string{})
	if err != nil {
		t.Fatalf("WriteFiles with no files: %v", err)
	}
	if len(written) != 0 {
		t.Errorf("WriteFiles with no files returned %v, want empty", written)
	}
}
