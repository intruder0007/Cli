# Authoring a capability plugin

A capability plugin mutates/adds files in an already-generated project
(e.g. `git-init`, `readme`, `github-actions-ci`). See
[docs/architecture/plugin-protocol.md](../architecture/plugin-protocol.md)
for the full wire contract this implements.

## In Go, using `sdk/go`

```go
package main

import "github.com/intruder0007/Cli/sdk/go/sdk"

type myCapability struct{}

func (myCapability) Apply(req sdk.ApplyRequest) (sdk.ApplyResponse, error) {
    // read req.TargetDir, req.ProjectName, req.Answers
    // write/modify files under req.TargetDir
    return sdk.ApplyResponse{
        FilesWritten: []string{"SOME_FILE"},
        NextSteps:    nil,
    }, nil
}

func main() {
    sdk.Serve(myCapability{})
}
```

## `plugin.json`

```json
{
  "protocolVersion": "1",
  "name": "my-capability",
  "version": "0.1.0",
  "kind": "capability",
  "displayName": "My Capability",
  "capabilityId": "my-capability",
  "entrypoint": "./my-capability"
}
```

Place the built binary and manifest under `plugins/builtin/my-capability/`
(first-party) or any local plugin directory the core is configured to
scan. No changes to `core` are required — this is the extensibility
contract in practice.

## Validate before shipping

Before packaging a plugin for release (or opening a PR that adds one),
run the pre-release check:

```sh
bootstrap plugins validate plugins/builtin/my-capability
```

This proves the directory's `plugin.json` parses and passes
`Manifest.Validate()`, and that the entrypoint binary actually spawns
and reports the same identity/protocol version as the manifest at the
`plugin.initialize` handshake — catching a stale or swapped binary
before anyone installs it. Exit 0 means valid.

## Non-Go plugins

The transport (subprocess + line-delimited JSON-RPC 2.0 over stdio) is
language-agnostic. A capability plugin in another language implements
`plugin.initialize`, `plugin.apply`, and `plugin.shutdown` directly per
[plugin-protocol.md](../architecture/plugin-protocol.md); there's no SDK
for it yet (see [roadmap.md](../architecture/roadmap.md)), but nothing in
the protocol privileges Go.
