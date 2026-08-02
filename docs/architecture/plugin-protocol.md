# Plugin protocol

Implements [ADR-0002](adr/0002-plugin-protocol.md). This is the contract
between `core` (the host) and any plugin (a template or a capability),
whether first-party (`templates/`, `plugins/builtin/`) or third-party.

## Discovery

At startup, `core/registry` scans `templates/`, `plugins/`, and any
user-configured local plugin directories for a `plugin.json` manifest in
each immediate subdirectory. There is no remote/network discovery in V1
(see [roadmap.md](roadmap.md)).

Since ADR-0008, a manifest that fails to parse or fails `Validate()`
(missing required field) is **skipped, not a hard failure** — one broken
third-party plugin can't take down discovery of everything else.
`Registry.DiscoverWithIssues()` reports what was skipped and why (`
lumo plugins list` prints these to stderr).

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
| `protocolVersion` | string | yes | Checked exactly against the core's version at `plugin.initialize` — see "Identity and protocol cross-check" below. |
| `name` | string | yes | Unique plugin identifier; cross-checked against the running process's own report at `plugin.initialize`. |
| `version` | string | yes | Plugin's own semver. |
| `kind` | `"template"` \| `"capability"` | yes | |
| `displayName` | string | yes | Shown in the wizard / `lumo plugins list`. |
| `projectType`, `language`, `framework` | string | templates only | Must match wizard answer values. |
| `supportedPlatforms` | []string | templates only, optional | GOOS values (`linux`/`darwin`/`windows`) this template supports. Empty means all platforms; `core/registry.ResolveTemplate` filters out non-matching platforms. See ADR-0009. |
| `capabilityId` | string | capabilities only | Stable id selected in the capabilities step. |
| `dependsOn` | []string | capabilities only, optional | Other selected capabilityIds that must be applied first (see "Capability ordering"). An entry naming a capability the user didn't select is ignored — plugins are never auto-installed. |
| `entrypoint` | string | yes | Path to the executable, relative to the manifest. |

`sdk.Manifest.Validate()` is the canonical rule set: `protocolVersion`,
`name`, `version`, `entrypoint` always required; `kind` must be
`"template"` or `"capability"`; templates additionally require
`projectType`/`language`/`framework`; capabilities additionally require
`capabilityId`. Both `core/registry` (discovering third-party manifests)
and `sdk.Serve()` (a plugin validating its own loaded manifest) apply it.

## Transport

The core spawns `entrypoint` as a subprocess. Requests and responses are
**JSON-RPC 2.0 objects, one per line** (newline-delimited), written to the
plugin's stdin and read from its stdout. Stderr is reserved for
human-readable plugin logs and is not parsed.

## Methods

### `plugin.initialize`

Request: `{ "protocolVersion": "1" }`
Response: `{ "ok": true, "manifest": { ...as above... } }`

**Identity and protocol cross-check** (ADR-0008): the host compares the
`manifest` in this response against the manifest it discovered on disk
for this plugin. `protocolVersion` must match exactly and `name` must
match, or the call fails (`plugin.ProtocolMismatchError` /
`plugin.IdentityMismatchError`) before `generate`/`apply` is ever called
— this catches a stale or swapped binary sitting at a plugin's
entrypoint path.

Every JSON-RPC call (including `initialize`) is bounded by
`Host.CallTimeout` (default 30s); a plugin that doesn't respond in time
is killed and the call fails with `plugin.TimeoutError`.

## Capability ordering

When multiple capabilities are selected, the host resolves *all* of them
(and the template) before invoking anything — a bad capability id fails
before the template or any earlier capability has written a file. The
resolved capabilities are then topologically sorted by `dependsOn`
(deterministic, stable — no `dependsOn` at all means the user's selection
order is preserved exactly); a cycle among selected capabilities'
`dependsOn` fails with `engine.CapabilityCycleError`.

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

Request: same shape as `plugin.generate` — `targetDir`, `projectName`,
and `answers` (`theme`/`projectType`/`language`/`framework`). The
`answers` map does **not** include which other capabilities were
selected; a capability plugin that needs to react to *another*
capability's presence has no way to today. `dependsOn` (above) only
controls execution order, not visibility. This is a real gap, not yet
needed by any V1 capability — see `roadmap.md`.

Response: `{ "filesWritten": [...], "filesModified": [...], "nextSteps": [...] }`

### `plugin.shutdown`

Request: `{}`. The core closes stdin after sending this and waits for the
process to exit (with a timeout, after which it is killed).

## Authoring a plugin

Go authors use `sdk/go`'s `sdk.Serve(yourPlugin)`, which implements this
transport and dispatches to whichever of `sdk.TemplatePlugin`
(`Generate`) and `sdk.CapabilityPlugin` (`Apply`) your type implements —
see [docs/plugins/authoring.md](../plugins/authoring.md) and
[docs/templates/authoring.md](../templates/authoring.md). Non-Go SDKs are
not built in V1, but any language can implement this protocol directly
from this document.

## Wire protocol stability

The exact bytes on the wire are pinned by golden transcript tests on
both sides, so a change to either implementation can't ship silently:

- **Host side** — `core/plugin/wire_test.go` builds
  `core/plugin/wiretest` (a real plugin binary served by `sdk/go`) and
  records the full lifecycle byte-for-byte against
  `core/plugin/testdata/wire-generate.golden` and
  `wire-apply.golden` (the exact requests the host emits, the exact
  responses a conforming SDK-served plugin returns).
- **Plugin side** — `sdk/go/sdk/wire_test.go` drives the protocol loop
  in-process against `sdk/go/sdk/testdata/serve-lifecycle.golden` (the
  same lifecycle, plus `plugin.apply`) and `serve-errors.golden` (the
  JSON-RPC error contract).

Together these pin: method names and request/response shapes (including
field names and serialization order), JSON-RPC id sequencing, the
`plugin.initialize` handshake and manifest reporting, the shutdown
sequence, and the JSON-RPC error codes (`-32700` parse error, `-32601`
unknown method / not implemented by this plugin kind, `-32602` invalid
params, `-32000` plugin failure). Note that Go's `encoding/json`
HTML-escapes non-ASCII-safe characters (`&` appears as `\u0026`); any
conforming JSON-RPC parser treats them as identical — third-party
implementations must accept both forms.

A wire-protocol change is always a breaking change (see
`api-compatibility.md`). When one is made deliberately, update this
document, the SDK bindings, and the goldens in the same commit:

```sh
go test ./core/plugin/ -run WireTranscript -update
go test ./sdk/go/sdk/ -run ServeWire -update
```
