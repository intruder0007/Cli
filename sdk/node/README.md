# Node.js SDK — design note, not implemented

Follow the [SDK architecture spec](../../docs/architecture/sdk-architecture.md)
(ADR-0014) — the language-neutral contract every SDK implements, and
the [plugin protocol](../../docs/architecture/plugin-protocol.md) it
binds to.

Status: **not implemented.** This directory is a design note, the same
shape ADR-0010 used for distribution wrappers before they were built.

## What building it requires

- An npm package `@intruder0007/cli-plugin-sdk` (name not reserved —
  placeholder), **zero npm dependencies**: `fs`, `readline`,
  `process`, `JSON` cover the entire transport (line-delimited JSON-RPC
  over stdio, exit on `plugin.shutdown`/EOF).
- Exports mirroring `sdk/go` one-to-one: `Manifest` (+`validate()`),
  `GenerateRequest`/`GenerateResponse`, `ApplyRequest`/
  `ApplyResponse`, `TemplatePlugin`/`CapabilityPlugin` (classes or
  duck-typed), `serve(plugin)`.
- A plugin is a single `.js` (or compiled) file plus `plugin.json`,
  exactly per `docs/plugins/authoring.md` and
  `docs/templates/authoring.md`.
- The reference behavior to match: `sdk/go`'s tests
  (`sdk/go/sdk/sdk_test.go`) are the behavioral spec.

## Compatibility constraints

- Binds to `protocolVersion: "1"`; no negotiation (see the SDK
  architecture spec's "Version negotiation").
- Must be implementable with `node` alone — a plugin built on this SDK
  must run with the same `node` the generated `templates/node-rest-api`
  already requires.
- An author porting a plugin from `sdk/go` changes only syntax — the
  acceptance test in `sdk-architecture.md`.
