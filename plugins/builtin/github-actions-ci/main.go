// Command github-actions-ci is a capability plugin: it adds a GitHub
// Actions workflow that runs `go build` and `go test` for the generated
// project.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	sdk "github.com/intruder0007/Lumo/sdk/go/sdk"
)

const workflow = `name: CI

on:
  push:
  pull_request:

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - run: go build ./...
      - run: go test ./...
`

type githubActionsCICapability struct{}

func (githubActionsCICapability) Apply(req sdk.ApplyRequest) (sdk.ApplyResponse, error) {
	// The workflow below is Go-only (setup-go, `go build`, `go test`).
	// Fail loudly for other languages rather than write a workflow that
	// can never be green in CI — the engine generates the template first,
	// so this error surfaces after the project exists but before the
	// workflow file is written.
	if lang := req.Answers["language"]; lang != "" && lang != "go" {
		return sdk.ApplyResponse{}, fmt.Errorf("github-actions-ci only supports Go projects (got language %q); pick a different capability or drop it", lang)
	}

	dir := filepath.Join(req.TargetDir, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return sdk.ApplyResponse{}, fmt.Errorf("creating .github/workflows: %w", err)
	}

	path := filepath.Join(dir, "ci.yml")
	if err := os.WriteFile(path, []byte(workflow), 0o644); err != nil {
		return sdk.ApplyResponse{}, fmt.Errorf("writing ci.yml: %w", err)
	}

	return sdk.ApplyResponse{
		FilesWritten: []string{".github/workflows/ci.yml"},
	}, nil
}

func main() {
	sdk.Serve(githubActionsCICapability{})
}
