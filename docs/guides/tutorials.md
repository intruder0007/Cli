# Tutorials

Three end-to-end walkthroughs. Tutorials 1 and 2 build real plugins
with `sdk/go`; tutorial 3 implements the wire protocol by hand in any
language, using the golden transcripts as test fixtures. All three
target the same manifest fields, behaviors, and protocol rules — an
author switching languages or approaches doesn't relearn semantics
(see [`sdk-architecture.md`](../architecture/sdk-architecture.md)).

## Tutorial 1: a capability plugin in Go

We'll build a `license` capability that writes an `LICENSE` file into
a generated project, then validate and use it.

### 1. Set up the module

```sh
mkdir license-plugin && cd license-plugin
go mod init license-plugin
go get github.com/intruder0007/Cli/sdk/go@latest
```

(The SDK is published as a normal Go module; within this repository
the `go.work` workspace resolves it in place.)

### 2. Implement the plugin

`main.go`:

```go
package main

import (
    "os"
    "path/filepath"

    "github.com/intruder0007/Cli/sdk/go/sdk"
)

type license struct{}

func (license) Apply(req sdk.ApplyRequest) (sdk.ApplyResponse, error) {
    path := filepath.Join(req.TargetDir, "LICENSE")
    if err := os.WriteFile(path, []byte("MIT License\n"), 0o644); err != nil {
        return sdk.ApplyResponse{}, err
    }
    return sdk.ApplyResponse{FilesWritten: []string{"LICENSE"}}, nil
}

func main() {
    sdk.Serve(license{})
}
```

The `answers` map carries the wizard's theme/projectType/language/
framework — read it if your capability needs it. Return an error and
the host reports it as a `-32000` JSON-RPC error; no files should be
partially written if you can avoid it.

### 3. Add the manifest

`plugin.json`:

```json
{
  "protocolVersion": "1",
  "name": "license",
  "version": "0.1.0",
  "kind": "capability",
  "displayName": "License",
  "capabilityId": "license",
  "entrypoint": "./license"
}
```

`protocolVersion` must equal `sdk.ProtocolVersion`; `entrypoint` is
relative to the manifest and must match the built binary name.

### 4. Build and validate

```sh
go build -o license .
bootstrap plugins validate .
```

Exit 0 means the manifest is valid **and** the binary spawns and
passes the identity/protocol cross-check — a stale or swapped binary
fails here, before anyone installs it.

### 5. Ship it

Place the binary and manifest under `plugins/builtin/license/`
(first-party) or a local plugin directory the core is configured to
scan. That's the whole extensibility contract — no core changes, and
the plugin is just one entry in `bootstrap plugins list`.

## Tutorial 2: a template plugin in Go

Same shape, but implementing `TemplatePlugin` and scaffolding files
into `TargetDir`.

### 1–2. Set up and implement

```sh
mkdir hello-template && cd hello-template
go mod init hello-template
go get github.com/intruder0007/Cli/sdk/go@latest
```

`main.go`:

```go
package main

import (
    "os"
    "path/filepath"

    "github.com/intruder0007/Cli/sdk/go/sdk"
)

type hello struct{}

func (hello) Generate(req sdk.GenerateRequest) (sdk.GenerateResponse, error) {
    main := filepath.Join(req.TargetDir, "main.go")
    if err := os.WriteFile(main, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
        return sdk.GenerateResponse{}, err
    }
    return sdk.GenerateResponse{
        FilesWritten: []string{"main.go"},
        NextSteps:    []string{"cd " + req.ProjectName + " && go run ."},
    }, nil
}

func main() {
    sdk.Serve(hello{})
}
```

### 3. Manifest

Templates require `projectType`, `language`, and `framework`, and those
values must match the wizard's answer lists exactly
(`backend-service`/`web-app`/`cli-tool`/`library`, `go`/`node`/
`typescript`/`python`/`rust`, `rest-api`/`http-api`/`grpc`/`graphql`
— a template whose values are invented is never offered; see
`cli/internal/prompt/prompt.go`). This "hello" template targets
`backend-service` + `go` + `http-api` (a combination no shipped
template covers yet):

```json
{
  "protocolVersion": "1",
  "name": "hello",
  "version": "0.1.0",
  "kind": "template",
  "displayName": "Hello",
  "projectType": "backend-service",
  "language": "go",
  "framework": "http-api",
  "entrypoint": "./hello"
}
```

### 4. Build, validate, ship

```sh
go build -o hello .
bootstrap plugins validate .
```

Then place it under `templates/hello/` (or a configured local plugin
directory). `bootstrap new` will offer it when the wizard's answers
match its `projectType`/`language`/`framework`.

## Tutorial 3: the protocol by hand, in any language

No SDK for your language yet? The transport is a subprocess speaking
line-delimited JSON-RPC 2.0 over stdio — implement four methods and
you're compatible. The golden transcripts in the repo are your test
fixtures: `sdk/go/sdk/testdata/serve-lifecycle.golden` is the exact
conversation a conforming plugin must produce.

### The loop

```text
read one line from stdin
  -> {"jsonrpc":"2.0","id":1,"method":"plugin.initialize",
      "params":{"protocolVersion":"1"}}
  <- {"jsonrpc":"2.0","id":1,"result":{"ok":true,
      "manifest":{... your plugin.json, verbatim ...}}}
  -> {"jsonrpc":"2.0","id":2,"method":"plugin.generate",
      "params":{"targetDir":"/abs/path","projectName":"demo",
      "answers":{...}}}
  <- {"jsonrpc":"2.0","id":2,"result":{"filesWritten":[...]}}
  -> {"jsonrpc":"2.0","id":3,"method":"plugin.shutdown","params":{}}
  <- {"jsonrpc":"2.0","id":3,"result":{"ok":true}}
exit
```

Rules:

- Exactly one JSON object per line, `\n`-terminated. Parse the whole
  request before answering (the host sends nothing until you respond —
  this is a strict request/response protocol, one call at a time).
- `plugin.initialize`'s response must contain your **own** loaded
  manifest (`protocolVersion` and `name` are cross-checked against
  the on-disk one — misreporting them fails the call).
- Errors are JSON-RPC 2.0 error objects: `-32700` parse, `-32601`
  unknown method, `-32602` invalid params, `-32000` your own failure.
- Accept both `\u0026` and `&` forms of escaped characters — Go's
  encoder HTML-escapes; a conforming JSON parser treats them as
  identical.
- Log to stderr, never stdout — stdout is protocol traffic.
- On `plugin.shutdown`, answer and exit. The host also closes stdin
  when it's done; EOF is a clean exit.

### Testing your implementation

Run your binary through `bootstrap plugins validate <dir>` (manifest +
handshake check), then generate a project and compare your output to
`core/plugin/testdata/wire-generate.golden` — the host emits exactly
those request lines, and a correct implementation answers exactly
those response lines.
