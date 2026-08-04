package prompt

import (
	"os"
	"path/filepath"
	"testing"
)

// The target-resolution rules are shared between the wizard's confirm
// screen and the non-interactive `new` path (both call
// ResolveTargetPath), so the bare-name / path / ~-expansion behavior is
// pinned here once.

func TestResolveTargetPathBareName(t *testing.T) {
	dir, name, err := ResolveTargetPath("my-app")
	if err != nil {
		t.Fatalf("bare name: unexpected error: %v", err)
	}
	if name != "my-app" {
		t.Errorf("name = %q, want %q", name, "my-app")
	}
	cwd, _ := os.Getwd()
	want := filepath.Join(cwd, "my-app")
	if dir != want {
		t.Errorf("dir = %q, want %q", dir, want)
	}
}

func TestResolveTargetPathRelative(t *testing.T) {
	dir, name, err := ResolveTargetPath("./a/b/app")
	if err != nil {
		t.Fatalf("relative path: unexpected error: %v", err)
	}
	if name != "app" {
		t.Errorf("name = %q, want %q", name, "app")
	}
	cwd, _ := os.Getwd()
	want := filepath.Join(cwd, "a", "b", "app")
	if dir != want {
		t.Errorf("dir = %q, want %q", dir, want)
	}
}

func TestResolveTargetPathAbsolute(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "proj")
	dir, name, err := ResolveTargetPath(abs)
	if err != nil {
		t.Fatalf("absolute path: unexpected error: %v", err)
	}
	if name != "proj" {
		t.Errorf("name = %q, want %q", name, "proj")
	}
	if dir != abs {
		t.Errorf("dir = %q, want %q", dir, abs)
	}
}

func TestResolveTargetPathHomeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	dir, name, err := ResolveTargetPath("~/code/app")
	if err != nil {
		t.Fatalf("~ expansion: unexpected error: %v", err)
	}
	if name != "app" {
		t.Errorf("name = %q, want %q", name, "app")
	}
	want := filepath.Join(home, "code", "app")
	if dir != want {
		t.Errorf("dir = %q, want %q", dir, want)
	}
}

func TestResolveTargetPathRejectsNonProjectNames(t *testing.T) {
	for _, in := range []string{"", ".", "..", "~", "~\\", "~/", "a/.."} {
		if _, _, err := ResolveTargetPath(in); err == nil {
			t.Errorf("ResolveTargetPath(%q): expected error, got none", in)
		}
	}
}

func TestIsPathLike(t *testing.T) {
	for _, in := range []string{"./x", "a/b", "..\\x", "~/code", "/abs/path", `C:\code\app`} {
		if !IsPathLike(in) {
			t.Errorf("IsPathLike(%q) = false, want true", in)
		}
	}
	for _, in := range []string{"my-app", "my_app2", "a-b.c"} {
		if IsPathLike(in) {
			t.Errorf("IsPathLike(%q) = true, want false", in)
		}
	}
}
