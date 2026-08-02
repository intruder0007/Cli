// Command wiretest is a scripted plugin responder for the wire-protocol
// golden transcript tests (core/plugin/wire_test.go). It implements both
// SDK interfaces with fixed, deterministic responses so the exact bytes
// on the wire can be pinned byte-for-byte. Not a real template or
// capability — test infrastructure only.
package main

import (
	"github.com/intruder0007/Lumo/sdk/go/sdk"
)

type plugin struct{}

func (plugin) Generate(sdk.GenerateRequest) (sdk.GenerateResponse, error) {
	return sdk.GenerateResponse{
		FilesWritten: []string{"go.mod", "main.go"},
		NextSteps:    []string{"cd new-project && go run ."},
	}, nil
}

func (plugin) Apply(sdk.ApplyRequest) (sdk.ApplyResponse, error) {
	return sdk.ApplyResponse{
		FilesWritten:  []string{"README.md", "Makefile"},
		FilesModified: []string{".gitignore"},
		NextSteps:     []string{"git add -A"},
	}, nil
}

func main() {
	sdk.Serve(plugin{})
}
