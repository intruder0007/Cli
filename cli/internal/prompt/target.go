package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// target.go resolves the user-supplied project-location string into the
// absolute target directory and the project name (the path's final
// component), so `lumo new ./a/b/app`, `lumo new /abs/path/app`, and a
// bare `lumo new my-app` all generate at the right place. This is the
// CLI's shared target-resolution rule: the wizard's confirm screen, the
// non-interactive `new` path, and the pre-run guard all use it.

// ResolveTargetPath splits a `lumo new` positional into the absolute
// target directory and the project name (the path's final component),
// so `lumo new ./a/b/app` and `lumo new /abs/path/app` generate at an
// explicit location instead of forcing a bare name into the CWD. A bare
// name (no separators, not "." or "..") targets the CWD, matching the
// historical behavior. A leading "~" (or "~\") expands to the user's
// home directory. The name is returned unchanged so Answers.Validate
// still rejects bad names with its usual message.
func ResolveTargetPath(positional string) (targetDir, projectName string, err error) {
	if positional == "" {
		return "", "", fmt.Errorf("project name is required")
	}
	p := positional
	if p == "~" || p == "~/" || p == "~\\" || p == "." || p == ".." || p == "" {
		return "", "", fmt.Errorf("target path %q does not name a project directory", positional)
	}
	if strings.HasPrefix(p, "~") {
		home, hErr := os.UserHomeDir()
		if hErr != nil {
			return "", "", fmt.Errorf("expanding ~: %w", hErr)
		}
		if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
			raw := p[2:] // drop the ~ and the separator
			p = filepath.Join(home, filepath.FromSlash(raw))
		} else {
			// "~foo" is a literal name, not a home expansion.
			p = positional
		}
	}
	name := filepath.Base(p)
	dir := filepath.Dir(p)
	if name == "." || name == ".." || name == "" || name == "/" || name == "\\" || name == string(filepath.Separator) {
		return "", "", fmt.Errorf("target path %q does not name a project directory", positional)
	}
	abs, err := filepath.Abs(filepath.Join(dir, name))
	if err != nil {
		return "", "", err
	}
	return abs, name, nil
}

// IsPathLike reports whether a wizard name input looks like an explicit
// path (relative/absolute/~) rather than a bare project name. The
// confirmation screen shows the resolved absolute target for paths and
// the bare name for bare names (design-spec §2.2 rule 4).
func IsPathLike(input string) bool {
	if strings.HasPrefix(input, "~") {
		return true
	}
	if strings.Contains(input, "/") || strings.Contains(input, "\\") {
		return true
	}
	return filepath.IsAbs(input) || strings.Contains(input, ":")
}
