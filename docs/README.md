# Documentation

The entry point for the project's documentation. Everything here is
written for two audiences: **users** of the CLI and **extenders** of the
platform (plugin/template authors, SDK users, wrapper maintainers).

## Getting started

- [`README.md`](../README.md) — what Lumo is, principles, installation,
  and a quick start.
- [`docs/cli/usage.md`](cli/usage.md) — every command, flag, and exit
  code, plus the interactive/non-interactive paths.

## Architecture (design docs + ADRs)

- [`docs/architecture/overview.md`](architecture/overview.md) — the
  subsystems and their boundaries.
- [`docs/architecture/plugin-protocol.md`](architecture/plugin-protocol.md) —
  the wire protocol between the host and plugins (Stable).
- [`docs/architecture/distribution-protocol.md`](architecture/distribution-protocol.md) —
  the wrapper contract every distribution method implements (Stable).
- [`docs/architecture/api-compatibility.md`](architecture/api-compatibility.md) —
  which surfaces are Stable and what changing them requires.
- [`docs/architecture/sdk-architecture.md`](architecture/sdk-architecture.md) —
  the language-neutral spec every SDK implementation follows.
- [`docs/architecture/codebase-audit.md`](architecture/codebase-audit.md) —
  the Phase F written audit: findings, fixes, verified-correct
  surfaces, and residual items.
- [`docs/architecture/roadmap.md`](architecture/roadmap.md) — what's
  shipped, deferred, and planned.
- [`docs/architecture/adr/`](architecture/adr/) — the decision records,
  numbered `0001`–`0015`, each statused Accepted.

## Guides

- [`docs/guides/api-reference.md`](guides/api-reference.md) — the four
  Stable surfaces, enumerated: wire protocol, `sdk/go`, CLI commands,
  and the distribution wrapper contract.
- [`docs/guides/versioning.md`](guides/versioning.md) — how releases,
  surfaces, and `protocolVersion` are versioned, and what each kind of
  version bump means to consumers.
- [`docs/guides/releasing.md`](guides/releasing.md) — the release
  runbook: cutting a `vX.Y.Z` tag, the automatic GitHub release and
  npm publish, post-release verification, rollback/deprecation, and
  long-term maintenance.
- [`docs/guides/migration-guide.md`](guides/migration-guide.md) — every
  breaking change and how to move between releases.
- [`docs/guides/tutorials.md`](guides/tutorials.md) — end-to-end
  tutorials: a capability plugin, a template plugin, and a hand-written
  protocol implementation.

## Extending the platform

- [`docs/plugins/authoring.md`](plugins/authoring.md) — capability
  plugins (`git-init`, `readme`, `github-actions-ci` are examples).
- [`docs/templates/authoring.md`](templates/authoring.md) — template
  plugins (`go-rest-api`, `node-rest-api` are examples).
- [`sdk/go`](../sdk/go) — the Go SDK's package doc; the only SDK
  shipped today (design notes for the future Node, Python, Rust, and
  other SDKs live under [`sdk/`](../sdk/)).
- [`distribution/`](../distribution/README.md) — the wrappers
  (brew, scoop, winget, chocolatey, cargo, npm, pypi).
