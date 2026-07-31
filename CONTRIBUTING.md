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
implement the `sdk.Plugin` interface from `sdk/go`, add a `plugin.json`
manifest, and place it under `templates/<name>/` or
`plugins/builtin/<name>/`. No core code changes are required — that's the
whole point of plugin-first extensibility.

## Coding standards

- Go: `gofmt`-formatted, `go vet` clean, tests colocated with the code they
  cover (`_test.go`).
- Docs: Markdown, linted with `markdownlint`.
- Commit messages: short imperative summary line; conventional prefixes
  (`fix:`, `feat:`, `docs:`, `test:`, `chore:`) are encouraged but not
  enforced by CI.
