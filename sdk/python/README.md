# Python SDK — design note, not implemented

Follow the [SDK architecture spec](../../docs/architecture/sdk-architecture.md)
(ADR-0014) — the language-neutral contract every SDK implements, and
the [plugin protocol](../../docs/architecture/plugin-protocol.md) it
binds to.

Status: **not implemented.** This directory is a design note, the same
shape ADR-0010 used for distribution wrappers before they were built.

## What building it requires

- A PyPI package `lumo-plugin-sdk` (name not reserved —
  placeholder), **stdlib only**: `json`, `sys`, `os` cover the entire
  transport (line-delimited JSON-RPC over stdio, exit on
  `plugin.shutdown`/EOF). No `requests`, no `pydantic`.
- Exports mirroring `sdk/go` one-to-one: `Manifest` (+`validate()`),
  `GenerateRequest`/`GenerateResponse`, `ApplyRequest`/
  `ApplyResponse`, `TemplatePlugin`/`CapabilityPlugin` (protocol
  classes), `serve(plugin)`.
- A plugin is a single `.py` entrypoint plus `plugin.json`, exactly per
  `docs/plugins/authoring.md` and `docs/templates/authoring.md`.
- The reference behavior to match: `sdk/go`'s tests
  (`sdk/go/sdk/sdk_test.go`) are the behavioral spec.

## Compatibility constraints

- Binds to `protocolVersion: "1"`; no negotiation (see the SDK
  architecture spec's "Version negotiation").
- Must run on the project's own minimum supported Python
  (currently unstated — decide at implementation time, matching the
  `distribution/pypi` wrapper's `requires-python`).
- An author porting a plugin from `sdk/go` changes only syntax — the
  acceptance test in `sdk-architecture.md`.
