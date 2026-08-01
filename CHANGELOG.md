# Changelog

All notable changes to this project are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versioning
follows [Semantic Versioning](https://semver.org/).

See [ADR-0006](docs/architecture/adr/0006-release-process.md) for the
release process this file is part of, and [CONTRIBUTING.md](CONTRIBUTING.md)
for how to add an entry.

## [Unreleased]

### Added

- Distribution architecture design (ADR-0010): the wrapper contract
  every future npm/PyPI/Cargo/Homebrew/Scoop/Winget/Chocolatey package
  must follow, plus scaffolded `distribution/<ecosystem>/` design notes.
  No wrapper is implemented — design and structure only, per scope.

### Added

- Second template, `templates/node-rest-api` — a Node.js HTTP API
  service (zero npm dependencies), the first real proof of
  "cross-language." `node`/`http-api` are now real, selectable wizard
  options. See ADR-0009.
- `Manifest.SupportedPlatforms` (templates only) — `core/registry`
  filters template resolution by the current platform.
- Manifest validation (`sdk.Manifest.Validate()`) — required fields per
  kind, checked both on discovery and by plugins validating their own
  manifest. `bootstrap plugins list` now reports skipped/invalid
  manifests instead of silently ignoring them.
- Identity + protocol version cross-check at the `plugin.initialize`
  handshake — catches a stale or swapped plugin binary before
  `generate`/`apply` runs against it.
- Optional `dependsOn` capability manifest field; selected capabilities
  are topologically ordered by it (stable — no `dependsOn` at all
  preserves the user's selection order exactly, true for every V1
  capability).
- Typed errors across `core` (`registry`/`plugin`/`engine`) replacing
  plain strings; `cli`'s error screen now matches on error type first.
- Per-call timeout on plugin JSON-RPC calls (default 30s) — a hung
  plugin is killed rather than blocking the CLI forever.

See ADR-0008.

### Changed

- `engine.Run` now resolves the template and every selected capability
  before invoking any of them (fail-fast — a bad capability id no
  longer lets an earlier capability's side effects happen first).

- Arrow-key/space-select interactive wizard (falls back to plain
  numbered-list prompts automatically when stdin/stdout aren't real
  terminals — no non-interactive behavior change). See ADR-0007.
- Theme is now a registry (`RegisterTheme`), not hardcoded — the
  concrete extension point for future themes.
- Theme persistence: `bootstrap config get|set theme`; the interactive
  wizard now remembers your last-chosen theme.
- Redesigned success/error screens; error screen includes a recovery
  hint for a few known failure shapes.
- Startup banner and richer per-command help (`bootstrap help`,
  `bootstrap <command> -h`).

### Changed

- `cli` module now depends on `golang.org/x/term` (raw-mode terminal
  input) — isolated to `cli` only; `core`/`sdk/go`/`templates/*`/
  `plugins/*` remain dependency-free.

## [0.1.1] - 2026-07-31

### Fixed

- `bootstrap plugins list` showed every plugin twice when running a
  released binary from its own extracted directory (the normal usage) —
  `pluginDirs()` now deduplicates its candidate directories by absolute
  path.

## [0.1.0] - 2026-07-31

Initial release. Interactive/non-interactive CLI wizard (theme, project
type, language, framework, capabilities), generating a Go REST API
backend service (`templates/go-rest-api`) with three capability plugins
(`git-init`, `readme`, `github-actions-ci`), over a subprocess +
line-delimited JSON-RPC 2.0 plugin protocol (`sdk/go`).

[Unreleased]: https://github.com/intruder0007/Cli/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/intruder0007/Cli/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/intruder0007/Cli/releases/tag/v0.1.0
