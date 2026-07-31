# ADR-0008: Plugin architecture hardening — validation, cross-check, typed errors, dependency ordering

## Status

Accepted

## Context

The pre-work audit for this phase (fresh re-read of the whole codebase,
same as Phase 1's) found concrete gaps in the V1 plugin contract:
`Manifest.protocolVersion` was sent in the `plugin.initialize` handshake
but never actually checked against anything; a malformed `plugin.json`
silently unmarshaled into a zero-value struct instead of erroring;
`core/plugin.Host` discarded the plugin's own `initialize` response
entirely, so a stale or swapped binary sitting at a plugin's entrypoint
path would go undetected; capabilities had no dependency concept beyond
"apply in the order the user clicked them"; a capability-resolution
failure happened *after* the template (and any earlier capabilities)
already ran, so partial side effects could occur before the user learned
a later capability id was misspelled; every failure mode was a plain
`fmt.Errorf` string, so `cli`'s error screen could only pattern-match
text.

Per the mandate: strengthen the *local* plugin architecture — no remote
registry (still explicitly deferred, per `roadmap.md`).

## Decision

- **Manifest validation** (`sdk.Manifest.Validate()`): required fields
  per kind, checked both by `core/registry` (discovering third-party
  manifests) and by `sdk.Serve()` (a plugin validating its own loaded
  manifest — defense in depth).
- **Discovery is resilient, not fragile**: `registry.Discover()` skips
  (doesn't hard-fail on) a manifest that fails to parse or fails
  `Validate()`, so one broken third-party plugin can't take down
  discovery of everything else. `DiscoverWithIssues()` additionally
  surfaces what was skipped and why — `bootstrap plugins list` now
  prints these to stderr.
- **Identity + protocol cross-check**: `Host.Generate`/`Apply` now
  capture the plugin's own `plugin.initialize` response (previously
  discarded) and compare it against the manifest the registry
  discovered on disk — protocol version must match exactly, and the
  plugin's self-reported name must match. A mismatch fails with a typed
  error rather than silently proceeding to `generate`/`apply` against
  whatever process happens to be listening at that entrypoint path.
- **Typed errors** replace plain strings for every failure mode a caller
  might reasonably want to handle differently:
  `registry.TemplateNotFoundError`/`CapabilityNotFoundError`;
  `plugin.StartError`/`ProtocolMismatchError`/`IdentityMismatchError`/
  `TimeoutError`; `engine.CapabilityCycleError`. `cli`'s `suggestFix`
  now matches via `errors.As` first (works through `engine.Run`'s
  `%w`-wrapping), falling back to string matching only for the one
  remaining untyped case (`core/config`'s validation errors — out of
  scope here).
- **Fail-fast resolution**: `engine.Run` resolves the template *and
  every selected capability* — cheap manifest lookups, no subprocess
  spawned — before invoking anything. A bad capability id now fails
  before the template (or any earlier capability) has written a file.
- **Capability dependency ordering**: an optional `dependsOn []string`
  manifest field (capabilities only), naming other selected
  capabilities that must apply first. `engine.sortByDependencies` is a
  deterministic, stable (Kahn's-algorithm, original-order tie-break)
  topological sort with cycle detection. A `dependsOn` entry naming a
  capability the user didn't select is ignored — V1 still never
  auto-installs anything. None of the three shipped V1 capabilities
  declare any `dependsOn`, so this is purely additive: existing
  generation behavior is provably unchanged (the black-box integration
  test, untouched, still passes).
- **Call timeout**: `Host.CallTimeout` (default 30s) races the blocking
  JSON-RPC read against a timer, so a hung plugin can't block the CLI
  forever — the process is killed and a `TimeoutError` returned.
- **Testability via small interfaces**: `engine.Engine` now depends on
  `Resolver`/`Runner` interfaces (structurally satisfied by
  `*registry.Registry`/`*plugin.Host`, no behavior change for real
  usage), enabling the fail-fast and dependency-ordering tests to run
  against fakes with zero subprocess/filesystem overhead.

## Consequences

- `core/plugin.Host.Generate`/`Apply` gained two parameters
  (`expectedName`, `expectedProtocolVersion`) — an internal API change
  within the `core` module (not a wire-protocol change); the one caller,
  `engine.Run`, was updated in the same change.
- `core/registry.Discover()`'s behavior changes slightly: previously a
  malformed manifest wasn't validated at all (would-be discovered as a
  broken `Plugin{}`); now it's silently skipped, with the reason
  available via `DiscoverWithIssues()`. This is strictly safer, but
  worth calling out as a behavior change, not just an addition.
- 16 new unit tests, all fast (no subprocess, no real plugin discovery)
  — manifest validation, discovery resilience, typed not-found errors,
  dependency-sort correctness (edges/stability/cycles/ignored
  unselected deps), fail-fast (proven via a fake `Runner` that records
  zero calls when resolution fails), the protocol/identity cross-check
  and call-timeout (proven via a `session` constructed directly around
  canned JSON-RPC responses / a never-written-to pipe, no real
  subprocess).
- A structured error taxonomy that plugins themselves can *return*
  (rather than just ones the host detects) is still out of scope — that
  would need a wire-protocol change (ADR-0002), not just a host-side
  one, and is a candidate for a future ADR if a concrete need arises.
