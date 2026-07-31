# Authoring a template plugin

A template plugin generates a brand-new project from scratch (V1's only
one: `templates/go-rest-api`). See
[docs/architecture/plugin-protocol.md](../architecture/plugin-protocol.md)
for the full wire contract this implements.

## In Go, using `sdk/go`

```go
package main

import "github.com/intruder0007/Cli/sdk/go/sdk"

type myTemplate struct{}

func (myTemplate) Generate(req sdk.GenerateRequest) (sdk.GenerateResponse, error) {
    // write files under req.TargetDir using req.ProjectName / req.Answers
    return sdk.GenerateResponse{
        FilesWritten: []string{"go.mod", "main.go"},
        NextSteps:    []string{"cd " + req.ProjectName + " && go run ."},
    }, nil
}

func main() {
    sdk.Serve(myTemplate{})
}
```

## `plugin.json`

```json
{
  "protocolVersion": "1",
  "name": "my-template",
  "version": "0.1.0",
  "kind": "template",
  "displayName": "My Template",
  "projectType": "backend-service",
  "language": "go",
  "framework": "my-framework",
  "entrypoint": "./my-template"
}
```

`projectType`/`language`/`framework` must match the values the wizard
uses so `core/registry` can resolve this template from user answers.

## Quality bar

A template must produce a project that **actually builds and passes its
own tests** — see [ADR-0004](../architecture/adr/0004-v1-scope.md). For a
Go template, that means the generated `go.mod`/`go build`/`go test` must
succeed standalone in the generated directory, independent of this
repository's `go.work`.

Place the built binary and manifest under `templates/my-template/`. No
changes to `core` are required.
