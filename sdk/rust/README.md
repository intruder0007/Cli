# Rust SDK — design note, not implemented

Follow the [SDK architecture spec](../../docs/architecture/sdk-architecture.md)
(ADR-0014) — the language-neutral contract every SDK implements, and
the [plugin protocol](../../docs/architecture/plugin-protocol.md) it
binds to.

Status: **not implemented.** This directory is a design note, the same
shape ADR-0010 used for distribution wrappers before they were built.

## What building it requires

- A crates.io package `bootstrap-cli-plugin-sdk` (name not reserved —
  placeholder), **zero dependencies**: the Rust standard library covers
  the entire transport (line-delimited JSON-RPC over stdio, exit on
  `plugin.shutdown`/EOF) — `std::io` + `serde`-free hand-rolled JSON is
  the honest reading of the project-wide "stdlib-only" convention, so
  weigh a single well-chosen dependency against that convention at
  implementation time (the distribution `build.rs` precedent shells out
  to system tools rather than adding crates).
- Exports mirroring `sdk/go` one-to-one: `Manifest` (+`validate()`),
  `GenerateRequest`/`GenerateResponse`, `ApplyRequest`/
  `ApplyResponse`, `TemplatePlugin`/`CapabilityPlugin` traits,
  `serve(plugin)`.
- A plugin is a single binary plus `plugin.json`, exactly per
  `docs/plugins/authoring.md` and `docs/templates/authoring.md`.
- The reference behavior to match: `sdk/go`'s tests
  (`sdk/go/sdk/sdk_test.go`) are the behavioral spec.

## Compatibility constraints

- Binds to `protocolVersion: "1"`; no negotiation (see the SDK
  architecture spec's "Version negotiation").
- The plugin binary must be statically buildable the way the core
  itself is (ADR-0001) — an SDK that forces dynamic linking would
  break the "single file per plugin" distribution model.
- An author porting a plugin from `sdk/go` changes only syntax — the
  acceptance test in `sdk-architecture.md`.
