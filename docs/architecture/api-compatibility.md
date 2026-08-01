# Public API compatibility

Implements [ADR-0013](adr/0013-api-compatibility.md). This is the
single source of truth for what counts as a public API in Cli, what
stability each surface is guaranteed at, and what versioning rules
apply when any of them changes. Contributors extending the platform
must be able to rely on exactly what is written here.

## What counts as a public API

| Surface | Location | Stability | Since |
|---|---|---|---|
| Plugin wire protocol | `docs/architecture/plugin-protocol.md` | **Stable** | v0.2.0 (ADR-0008 hardening) |
| Go SDK (`sdk/go`) | `github.com/intruder0007/Cli/sdk/go/sdk` | **Stable** | v0.3.0 |
| CLI command surface | `bootstrap new/plugins/config/doctor/version/help` — `docs/cli/usage.md` | **Stable** | v0.2.0 |
| Distribution wrapper contract | `docs/architecture/distribution-protocol.md` | **Stable** | v0.3.0 |
| Theme API | `cli/internal/prompt` (`RegisterTheme`, `GetTheme`, `Theme`) | **Experimental (internal)** | v0.2.0 |
| `core` packages (`registry`, `plugin`, `engine`, `config`, `diag`) | Go modules `core/*` | **Experimental (internal)** | v0.1.0 |
| `cli/internal/*` (anything under `cli/internal`, including `embedded`) | Go `internal` packages | **Internal only** — not public by definition | v0.1.0 |

"Experimental (internal)" means: reachable in source, used by the CLI
itself, but **not yet a promise to external contributors**. These
surfaces can change without a major version bump. A surface moves to
**Stable** only via an ADR that commits to its shape — see the
lifecycle below.

## Lifecycle stages

Every public surface is in exactly one of these stages:

1. **Experimental** — exists and is exercised, but shape is not
   guaranteed. Changes don't require a major version. Explicitly
   marked "Experimental" in docs.
2. **Stable** — committed contract. Breaking changes require a major
   version bump and a deprecation period (below). This is where
   external contributors should build.
3. **Deprecated** — still works and is still honored, but is
   documented as superseded and scheduled for removal. Deprecated
   surfaces keep working for at least one full minor release after
   the deprecation notice.
4. **Removed** — no longer present. Removal happens only after a
   Deprecated period, in a major version.

Promotion (Experimental → Stable) and demotion both require an ADR,
so the decision is recorded and reviewable. Deprecation and removal
are announced in `CHANGELOG.md` at the release where they take effect.

## Versioning rules

The project uses semver for releases (ADR-0006). The rules below bind
releases to the surfaces above:

- **Major** (`v1.0.0`, `v2.0.0`): any breaking change to a **Stable**
  surface — the wire protocol, `sdk/go`'s exported API, the CLI
  command surface, or the distribution wrapper contract.
- **Minor** (`v0.3.1 → v0.3.2`, `v1.1.0`): additive changes to Stable
  surfaces (new optional manifest field, new SDK helper, new command)
  and any change to Experimental surfaces.
- **Patch** (`v0.3.1`): bug fixes that don't change documented
  behavior.

While the project is at `v0.x`, the convention is the common Go
ecosystem one: **a breaking change to a Stable surface still bumps
the minor version** (e.g. `v0.3.0` → `v0.4.0`), never a patch, and is
still called out in the changelog — but formal major-version
protection applies from `v1.0.0` onward.

### The wire protocol's own version

Independent of the CLI's semver, `plugin.json`'s `protocolVersion`
field and `sdk.ProtocolVersion` (both `"1"` today) identify the wire
protocol. Rules:

- The host (core) supports **exactly one** protocol version per
  release — the one it was built against.
- A plugin whose `protocolVersion` differs is rejected at the
  `plugin.initialize` handshake with `plugin.ProtocolMismatchError`
  (ADR-0008) — never silently misbehaved with.
- A wire-protocol change is always a **breaking change** to the
  plugin wire protocol surface and therefore requires the deprecation
  period + major version rules above. Version negotiation across two
  simultaneously-supported protocol versions is deferred (see
  `roadmap.md`).
- The exact bytes of the wire protocol are **enforced by golden
  transcript tests** on both sides of the wire (see "Wire protocol
  stability" in `plugin-protocol.md`): a host or SDK change that alters
  any method name, parameter or response field, id sequencing, the
  handshake, the shutdown sequence, or an error code fails CI until the
  goldens and this policy are updated together.

## What Stable guarantees, concretely

For the four Stable surfaces, the following are guaranteed once
released in a version:

- **Wire protocol**: request/response shapes, method names,
  manifest fields and their semantics, and the discovery rules in
  `plugin-protocol.md` keep working as documented. A plugin built
  against a Stable protocol version keeps working with any CLI whose
  `sdk.ProtocolVersion` matches.
- **`sdk/go`**: every exported identifier is a contract —
  types, fields, function signatures, and constants. Programs
  written against it keep compiling and behaving as documented.
  Internal implementation details (unexported types, the transport
  loop's internals) are not.
- **CLI commands**: flags, positional arguments, exit codes, and
  output lines documented in `docs/cli/usage.md` keep working. New
  flags and commands are additive; changing documented behavior is a
  breaking change.
- **Distribution wrappers**: the four-step contract in
  `distribution-protocol.md` (resolve platform → locate/fetch the
  archive → exec with stdio passthrough → forward argv/exit code)
  is binding on every wrapper and every release that ships one.

## How to change a Stable surface

1. Add an ADR describing the change, the migration path, and why the
   break is worth it.
2. Deprecate the old shape in docs and code (at least one full minor
   release).
3. Make the breaking change in the next major (or, pre-`v1.0.0`,
   the next minor) release, and document it in `CHANGELOG.md`'s
   "Changed"/"Removed" sections.
4. Provide a migration guide entry in `docs/` when the change is
   user-visible (see `docs/` for existing guides; new ones follow the
   same format).

## Additive changes that need no ADR

Anything that only adds to a Stable surface — a new optional manifest
field, a new SDK function, a new CLI flag, a new wrapper — is an
ordinary PR (still with tests and changelog entry). The ADR bar is
for changes that could break an existing consumer.
