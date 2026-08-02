# API reference

The four **Stable** surfaces (per
[`docs/architecture/api-compatibility.md`](../architecture/api-compatibility.md))
enumerated: what exists, where it lives, and its exact contract.
Experimental surfaces (`core` packages, the theme API) are documented
in source and are not listed here — they are not promised to external
consumers yet.

## 1. Plugin wire protocol (Stable)

The contract between the host (`core/plugin`) and any plugin process.
Full spec: [`docs/architecture/plugin-protocol.md`](../architecture/plugin-protocol.md).
The exact bytes are pinned by golden transcripts
(`core/plugin/testdata/wire-*.golden`, `sdk/go/sdk/testdata/*.golden`).

| Method | Params | Result |
|---|---|---|
| `plugin.initialize` | `{"protocolVersion": "1"}` | `{"ok": true, "manifest": {...}}` — the plugin's *own* loaded manifest; the host cross-checks `protocolVersion` and `name` against the on-disk manifest (ADR-0008). |
| `plugin.generate` | `targetDir`, `projectName`, `answers` (map) | `filesWritten` ([]string), `nextSteps` ([]string, optional) — templates only. |
| `plugin.apply` | same as `plugin.generate` | `filesWritten`, `filesModified` (optional), `nextSteps` (optional) — capabilities only. |
| `plugin.shutdown` | `{}` | `{"ok": true}` — the host closes stdin and waits for exit. |

Framing: exactly one JSON-RPC 2.0 object per line, LF-terminated, on
stdio. Stderr is for human logs only. Every call is bounded by the
host's call timeout; a plugin that doesn't respond is killed.

Errors (JSON-RPC 2.0 error objects):

| Code | Meaning |
|---|---|
| `-32700` | Malformed request (parse error). |
| `-32601` | Unknown method, or a method the plugin's kind doesn't implement. |
| `-32602` | Invalid params (e.g. `targetDir` not a string). |
| `-32000` | The plugin's own error (mapped by the SDK). |

## 2. `sdk/go` (Stable)

Module `github.com/intruder0007/Cli/sdk/go`, package `sdk`. Every
exported identifier is a contract (see the package doc).

### Constants

```go
const ProtocolVersion = "1" // the wire protocol version served by this SDK
```

### Serving a plugin

```go
func Serve(plugin interface{})
```

Runs the transport loop until `plugin.shutdown` or EOF: reads the
manifest from `plugin.json` next to the executable, validates it, then
dispatches line-delimited JSON-RPC over stdio. `plugin` must implement
`TemplatePlugin`, `CapabilityPlugin`, or both.

### Interfaces

```go
type TemplatePlugin interface {
    Generate(GenerateRequest) (GenerateResponse, error)
}

type CapabilityPlugin interface {
    Apply(ApplyRequest) (ApplyResponse, error)
}
```

### Manifest

```go
type Manifest struct {
    ProtocolVersion     string   // required
    Name                string   // required — unique id, cross-checked at initialize
    Version             string   // required — the plugin's own semver
    Kind                string   // "template" | "capability" (required)
    DisplayName         string   // shown in the wizard / plugins list
    ProjectType         string   // templates only, required
    Language            string   // templates only, required
    Framework           string   // templates only, required
    CapabilityID        string   // capabilities only, required
    DependsOn           []string // capabilities only, optional — ordering among selected capabilities
    SupportedPlatforms  []string // templates only, optional — GOOS values
    Entrypoint          string   // required — executable path relative to the manifest
}

func (m Manifest) Validate() error // canonical rule set for plugin.json
```

### Requests and responses (protocol constants)

```go
type GenerateRequest struct {
    TargetDir   string
    ProjectName string
    Answers     map[string]string
}

type GenerateResponse struct {
    FilesWritten []string
    NextSteps    []string // optional
}

type ApplyRequest struct {
    TargetDir   string
    ProjectName string
    Answers     map[string]string
}

type ApplyResponse struct {
    FilesWritten  []string
    FilesModified []string // optional
    NextSteps     []string // optional
}
```

The JSON field names (`targetDir`, `projectName`, `answers`,
`filesWritten`, `filesModified`, `nextSteps`) are wire constants — they
are not design choices per SDK (see
[`sdk-architecture.md`](../architecture/sdk-architecture.md)).

## 3. CLI command surface (Stable)

Full details and flags: [`docs/cli/usage.md`](../cli/usage.md).

| Command | Contract |
|---|---|
| `bootstrap new` | Interactive wizard; non-interactive whenever a positional `name` or any flag is given — there is no `--non-interactive` flag, it's implicit (and `--answers <file>` is the scriptable path); `--verbose`/`-v` diagnostics. |
| `bootstrap plugins list` | Lists discovered template and capability plugins (name, kind, version). |
| `bootstrap plugins validate <plugin-dir>` | Pre-release check; exit 0 = valid, 1 = invalid. |
| `bootstrap config get theme` / `bootstrap config set theme <name>` | Theme persistence. |
| `bootstrap doctor` | Local health checks (plugin directory setup, wrapper setup). |
| `bootstrap version` | `bootstrap version <v> (go<ver>, <os>/<arch>)`. |
| `bootstrap help` / `bootstrap <command> -h` | Help. |

Exit codes and output lines documented in `usage.md` are part of this
surface.

## 4. Distribution wrapper contract (Stable)

Every wrapper (homebrew, scoop, winget, chocolatey, cargo, npm, pypi —
see [`distribution/`](../../distribution/README.md)) implements the
four-step contract in
[`docs/architecture/distribution-protocol.md`](../architecture/distribution-protocol.md):

1. Resolve the platform.
2. Locate/fetch the release archive.
3. Exec the embedded `bootstrap` binary with stdio passthrough.
4. Forward argv and the exit code.

The binary embeds its own plugin set as a fallback (ADR-0012), so a
wrapper only needs to deliver the binary itself.
