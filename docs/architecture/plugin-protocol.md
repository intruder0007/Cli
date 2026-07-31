# Plugin protocol

Implements [ADR-0002](adr/0002-plugin-protocol.md). This is the contract
between `core` (the host) and any plugin (a template or a capability),
whether first-party (`templates/`, `plugins/builtin/`) or third-party.

## Discovery

At startup, `core/registry` scans `templates/`, `plugins/`, and any
user-configured local plugin directories for a `plugin.json` manifest in
each immediate subdirectory. There is no remote/network discovery in V1
(see [roadmap.md](roadmap.md)).

## Manifest (`plugin.json`)

```json
{
  "protocolVersion": "1",
  "name": "go-rest-api",
  "version": "0.1.0",
  "kind": "template",
  "displayName": "Go REST API Service",
  "projectType": "backend-service",
  "language": "go",
  "framework": "rest-api",
  "entrypoint": "./go-rest-api"
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `protocolVersion` | string | yes | Must match the core's supported version(s). |
| `name` | string | yes | Unique plugin identifier. |
| `version` | string | yes | Plugin's own semver. |
| `kind` | `"template"` \| `"capability"` | yes | |
| `displayName` | string | yes | Shown in the wizard / `bootstrap plugins list`. |
| `projectType`, `language`, `framework` | string | templates only | Must match wizard answer values. |
| `capabilityId` | string | capabilities only | Stable id selected in the capabilities step. |
| `entrypoint` | string | yes | Path to the executable, relative to the manifest. |

## Transport

The core spawns `entrypoint` as a subprocess. Requests and responses are
**JSON-RPC 2.0 objects, one per line** (newline-delimited), written to the
plugin's stdin and read from its stdout. Stderr is reserved for
human-readable plugin logs and is not parsed.

## Methods

### `plugin.initialize`

Request: `{ "protocolVersion": "1" }`
Response: `{ "ok": true, "manifest": { ...as above... } }`

### `plugin.generate` (templates only)

Request:

```json
{
  "targetDir": "/abs/path/to/new-project",
  "projectName": "new-project",
  "answers": { "theme": "default", "projectType": "backend-service", "language": "go", "framework": "rest-api" }
}
```

Response: `{ "filesWritten": ["go.mod", "main.go", ...], "nextSteps": ["cd new-project && go run ."] }`

### `plugin.apply` (capabilities only)

Request: same shape as `plugin.generate`, plus the capability sees the
full `answers` map (including which other capabilities were selected, for
ordering-aware plugins).

Response: `{ "filesWritten": [...], "filesModified": [...], "nextSteps": [...] }`

### `plugin.shutdown`

Request: `{}`. The core closes stdin after sending this and waits for the
process to exit (with a timeout, after which it is killed).

## Authoring a plugin

Go authors use `sdk/go`'s `sdk.Serve(plugin sdk.Plugin)`, which implements
this transport and dispatches to a small `Generate`/`Apply` interface — see
[docs/plugins/authoring.md](../plugins/authoring.md) and
[docs/templates/authoring.md](../templates/authoring.md). Non-Go SDKs are
not built in V1, but any language can implement this protocol directly
from this document.
