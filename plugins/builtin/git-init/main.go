// Command git-init is a capability plugin: it initializes a Git
// repository in the generated project and creates an initial commit.
package main

import (
	"fmt"
	"os/exec"

	sdk "github.com/intruder0007/Cli/sdk/go/sdk"
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
	if err := run(req.TargetDir, "init"); err != nil {
		return sdk.ApplyResponse{}, err
	}
	if err := run(req.TargetDir, "add", "-A"); err != nil {
		return sdk.ApplyResponse{}, err
	}
	if err := run(req.TargetDir, "commit", "-m", "Initial commit", "--allow-empty"); err != nil {
		return sdk.ApplyResponse{}, err
	}

	return sdk.ApplyResponse{
		FilesWritten: []string{".git/"},
		NextSteps:    []string{"git log --oneline"},
	}, nil
}

func main() {
	sdk.Serve(gitInitCapability{})
}
