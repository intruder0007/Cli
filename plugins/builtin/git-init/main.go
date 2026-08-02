// Command git-init is a capability plugin: it initializes a Git
// repository in the generated project and creates an initial commit.
package main

import (
	"fmt"
	"os/exec"

	sdk "github.com/intruder0007/Lumo/sdk/go/sdk"
)

type gitInitCapability struct{}

func run(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %v: %w: %s", args, err, out)
	}
	return nil
}

func (gitInitCapability) Apply(req sdk.ApplyRequest) (sdk.ApplyResponse, error) {
	// Report exactly what was created: the .git repository directory
	// (its object store, index, and refs — hence the trailing slash).
	// The project's source files were written by the template plugin,
	// not here, so they must not be claimed as FilesWritten.
	if err := run(req.TargetDir, "init"); err != nil {
		return sdk.ApplyResponse{}, fmt.Errorf("git init: %w", err)
	}
	if err := run(req.TargetDir, "add", "-A"); err != nil {
		return sdk.ApplyResponse{}, fmt.Errorf("git add: %w", err)
	}
	if err := run(req.TargetDir, "commit", "-m", "Initial commit", "--allow-empty"); err != nil {
		return sdk.ApplyResponse{}, fmt.Errorf("git commit: %w", err)
	}

	return sdk.ApplyResponse{
		FilesWritten: []string{".git/"},
		NextSteps:    []string{"git log --oneline", "git status"},
	}, nil
}

func main() {
	sdk.Serve(gitInitCapability{})
}
