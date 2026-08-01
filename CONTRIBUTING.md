# Contributing

Thanks for considering a contribution. This document describes how the
repository is organized and how changes flow into `main`.

## Branch map

| Branch | Purpose |
|---|---|
| `main` | Integration branch. Releases are tagged here. **No direct pushes.** |
| `architecture` | ADRs, architecture docs, root project scaffolding |
| `cli` | The interactive/non-interactive CLI |
| `core` | Orchestration engine, plugin host, config model, registry |
| `sdk` | Plugin/template author SDKs (`sdk/go` first) |
| `templates` | Template plugins |
| `plugins` | Built-in capability plugins |
| `tests` | Integration/golden tests, CI workflow |

Each long-lived branch owns a subsystem and is an independently buildable Go
module (or set of modules). Cross-subsystem contracts live in `sdk/go`
(wire types) and are documented in
[`docs/architecture/plugin-protocol.md`](docs/architecture/plugin-protocol.md).

## Workflow

1. **Never commit directly to `main`.** Branch from the relevant long-lived
   branch (e.g. a topic branch off `cli` for CLI changes), or work directly
   on the long-lived branch for larger efforts.

   **Once a long-lived branch has been merged into `main`, treat it as
   stale rather than branching further topic work from it directly** —
   it won't have picked up whatever other long-lived branches merged into
   `main` afterward (V1's `cli` branch, for example, predates `core`,
   `sdk`, etc. all being on `main`). Branch new work from `main` instead;
   only branch from a long-lived branch if you've first confirmed it's
   actually up to date with `main`.
2. Open a pull request into the long-lived branch (for topic branches) or
   into `main` (for a long-lived branch that's ready to ship).
3. CI (`.github/workflows/ci.yml`) must pass: `go vet`, `go build`, `go test`
   across all modules, `markdownlint` on `docs/`, and the integration test.
4. At least one review is required before merging into `main`.

## ADRs

Any decision that changes a public interface, a subsystem boundary, the
plugin protocol, or is otherwise hard to reverse **requires an
Architecture Decision Record** in `docs/architecture/adr/`, following the
existing numbered files as a template (Status / Context / Decision /
Consequences). Open the ADR in the same PR as the change it justifies.

## Adding a template or capability plugin

See [`docs/plugins/authoring.md`](docs/plugins/authoring.md) and
[`docs/templates/authoring.md`](docs/templates/authoring.md). In short:
implement `sdk.TemplatePlugin` (a `Generate` method) and/or
`sdk.CapabilityPlugin` (an `Apply` method) from `sdk/go`, call
`sdk.Serve(yourPlugin)`, add a `plugin.json` manifest, and place it under
`templates/<name>/` or `plugins/builtin/<name>/`. No core code changes
are required — that's the whole point of plugin-first extensibility.

## Releases

See [ADR-0006](docs/architecture/adr/0006-release-process.md). In short:

- Add a bullet under `## [Unreleased]` in [`CHANGELOG.md`](CHANGELOG.md)
  for any user-visible change, in the same PR as the change.
- To cut a release: rename `[Unreleased]` to the new `[x.y.z] - YYYY-MM-DD`
  version, add a fresh empty `[Unreleased]` section above it, update the
  compare links at the bottom of the file, commit, then push a `vX.Y.Z`
  tag. Pushing the tag triggers `.github/workflows/release.yml`, which
  cross-compiles the CLI + all V1 plugins for every target and publishes
  a GitHub Release with checksums — no manual build/upload step.

## Coding standards

- Go: `gofmt`-formatted, `go vet` clean, tests colocated with the code they
  cover (`_test.go`).
- Docs: Markdown, linted with `markdownlint`.
- Commit messages: short imperative summary line; conventional prefixes
  (`fix:`, `feat:`, `docs:`, `test:`, `chore:`) are encouraged but not
  enforced by CI.
