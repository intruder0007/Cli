# Versioning guide

How this project versions things, and what each version number means
to you as a consumer — user, plugin author, or wrapper maintainer.

## Releases: semver (ADR-0006)

Releases are tagged `vX.Y.Z` on `main` (today: `v0.1.0` … `v0.3.0`).
The version is injected at build time via `-ldflags` from
`git describe --tags`, never hand-edited. `CHANGELOG.md` documents
every release with `Added`/`Changed`/`Fixed`/`Removed` sections.

## Surfaces: Stable vs Experimental (ADR-0013)

Changes are judged against the **four Stable surfaces**:

| Surface | Where |
|---|---|
| Plugin wire protocol | `docs/architecture/plugin-protocol.md` |
| `sdk/go` exported API | `sdk/go` |
| CLI command surface | `docs/cli/usage.md` |
| Distribution wrapper contract | `docs/architecture/distribution-protocol.md` |

Everything else — the `core` packages, the theme API — is
Experimental (internal) and can change at any minor release.

## What each bump means

| Bump | Guarantee |
|---|---|
| **Major** (`v2.0.0`) | Breaking change to a Stable surface. Requires the deprecation period + a migration guide entry (see below). |
| **Minor** (`v0.3.0 → v0.4.0`, `v1.1.0`) | Additive changes to Stable surfaces, and any change to Experimental surfaces. Breaking changes to Stable surfaces also land here while the project is at `v0.x` (Go ecosystem convention) — and are **always** called out in the changelog and migration guide. |
| **Patch** (`v0.3.1`) | Bug fixes that don't change documented behavior. |

## The wire protocol's own version

Independent of the release semver:

- `plugin.json`'s `protocolVersion` and `sdk.ProtocolVersion` are both
  `"1"` today, and identify the wire protocol.
- The host supports **exactly one** protocol version per release. A
  plugin whose manifest disagrees is rejected at the `plugin.initialize`
  handshake with `plugin.ProtocolMismatchError` — never silently
  misbehaved with.
- A wire-protocol change is always a breaking change, so it arrives
  with a major (or, at `v0.x`, a minor) bump **and** a migration-guide
  entry. Version negotiation across two simultaneously-supported
  versions is deferred (see `roadmap.md`).

## What plugin and template authors should do

1. **Pin `protocolVersion` to the version your SDK serves** — copy it
   from `sdk.ProtocolVersion`, never invent one.
2. **Treat `sdk/go`'s exported API as stable** — code written against
   a release keeps compiling and behaving against later minor and
   patch releases of the same major.
3. **Run `bootstrap plugins validate <dir>` before releasing** a
   plugin, so a stale or swapped binary never ships (see the authoring
   guides).
4. **Check the migration guide when upgrading** the CLI your plugin is
   tested against — a `Minor`/`Major` release may contain a
   user-visible breaking change that needs an action from you.

## Wrapper maintainers

The four-step contract
([`distribution-protocol.md`](../architecture/distribution-protocol.md))
is binding on every wrapper and every release. A release that changes
the contract is a breaking change (Major bump, or Minor at `v0.x`) and
is called out in the migration guide. Keep the wrapper's pinned version
at the latest release.

## Breaking-change lifecycle

1. An ADR describes the change, the migration path, and why the break
   is worth it.
2. The old shape is deprecated in docs and code for at least one full
   minor release.
3. The breaking change lands in the next major (or, pre-`v1.0.0`, the
   next minor) release, documented in `CHANGELOG.md`'s
   "Changed"/"Removed" sections.
4. A migration-guide entry lands in
   [`migration-guide.md`](migration-guide.md) when the change is
   user-visible.
