# Changelog

All notable changes to this project are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versioning
follows [Semantic Versioning](https://semver.org/).

See [ADR-0006](docs/architecture/adr/0006-release-process.md) for the
release process this file is part of, and [CONTRIBUTING.md](CONTRIBUTING.md)
for how to add an entry.

<!-- markdownlint-disable MD024 -->
<!-- Keep a Changelog's format reuses ### Added/Changed/Fixed under every
     version section by design; MD024 (no-duplicate-heading) would
     otherwise flag every version after the first. -->

## [Unreleased]

### Added

- Public API compatibility policy (ADR-0013 + `api-compatibility.md`):
  an explicit inventory of every public surface with its stability
  stage (Stable: wire protocol, `sdk/go`, CLI commands, distribution
  contract; Experimental: theme API, `core` packages), lifecycle
  stages (Experimental → Stable → Deprecated → Removed), and semver
  rules binding releases to surfaces. `sdk/go`'s package doc now
  declares its Stable status.
- SDK foundation (ADR-0014 + `sdk-architecture.md`): the
  language-neutral SDK specification every future SDK implements —
  four abstractions (Manifest, Plugin, Requests/Responses, Serve),
  exact protocol bindings, package layout, transport rules,
  compatibility strategy (SDKs are additive; `sdk/go` is the
  reference implementation), and version negotiation (none today,
  multi-version sketched). Design notes added under
  `sdk/{node,python,rust,future}/`; no SDK beyond `sdk/go` is
  implemented, per scope.
- `bootstrap plugins validate <dir>` — pre-release extension check:
  validates a plugin directory's manifest (`Manifest.Validate()`) and
  proves the entrypoint binary spawns and passes the
  `plugin.initialize` identity/protocol cross-check against the
  on-disk manifest (the same fail-fast surface `new` uses, ADR-0008),
  without generating anything. Exit 0 = valid. Backed by new
  `core/registry.LoadPluginDir` and `core/plugin.Host.Validate`
  (additive, Experimental surface per ADR-0013).
- Wire-protocol compatibility tests (Phase D): byte-exact golden
  transcripts pin the full JSON-RPC lifecycle — `plugin.initialize`,
  `plugin.generate`/`plugin.apply`, `plugin.shutdown` — on both sides
  of the wire (`core/plugin/testdata/wire-generate.golden`,
  `wire-apply.golden` from a real SDK-served responder binary;
  `sdk/go/sdk/testdata/serve-lifecycle.golden`, `serve-errors.golden`
  for the JSON-RPC error contract). Regenerable via `-update`.
  `sdk.Serve`'s protocol loop was refactored into an unexported
  `serveWithIO(reader, writer)` for in-process byte-exact testing;
  public API and behavior unchanged. See "Wire protocol stability" in
  `plugin-protocol.md`.

### Changed

- All 7 distribution wrappers repointed at the real `v0.3.0` release
  assets (package versions, download URLs, `extract_dir`, and
  checksums from the published `SHA256SUMS.txt`), re-verified against
  them in `distribution-verify.yml` CI. None are published yet.

## [0.3.0] - 2026-08-02

### Added

- Distribution wrappers for 7 of 8 planned ecosystems (npm, PyPI,
  Cargo, Homebrew, Scoop, Winget, Chocolatey) under `distribution/` —
  built and verified (locally and/or via new
  `.github/workflows/distribution-verify.yml`, one job per ecosystem)
  against real published `v0.2.0` release assets. None are published to
  any real registry — that needs either a credential or a PR to a
  third-party repo, both separate, explicitly-confirmed future steps.
  See each ecosystem's `distribution/<name>/README.md`.
- Universal install architecture (ADR-0012): the `cli` binary now embeds
  the V1 plugin set at build time and self-extracts it to a
  version-scoped cache directory whenever no sibling
  `templates/`/`plugins/builtin/` directories can be found — closing
  the `go install` gap ADR-0010 had left open. `go install
  github.com/intruder0007/Cli/cli@latest` now works out of the box.
- `install.sh`/`install.ps1` — one-line install scripts that resolve
  platform, verify the release archive's checksum, and put just the
  `bootstrap` binary on `PATH`.
- `bootstrap doctor` now reports whether this binary has embedded
  plugin assets and whether the embedded fallback is currently serving
  plugins for the run.

### Changed

- `Makefile`'s `build` target and `.github/workflows/release.yml` now
  stage built plugin binaries into `cli/internal/embedded/assets/`
  before building `cli`, so each build's binary embeds its own
  platform's plugin set (see ADR-0012's "Build-order coupling").

## [0.2.0] - 2026-08-01

### Added

- `core/diag` — a minimal `Logger` seam shared by `core/engine` and
  `core/plugin`; nil-safe (defaults to a no-op) and opt-in.
- `bootstrap new --verbose`/`-v` — prints diagnostic logging (plugin
  spawn, handshake result, timing, file counts) to stderr as a run
  progresses.
- `bootstrap doctor` — local health check: verifies plugin directories
  resolve and every discovered manifest is valid, with a pass/fail
  summary and recovery hint.
- `bootstrap version` now prints the Go runtime version and OS/arch
  alongside the CLI's semver string.
- Success screen now shows a one-line project summary (template used,
  file count, capabilities applied) via new `engine.Summary.Template`/
  `CapabilitiesApplied` fields. See ADR-0011.
- Distribution architecture design (ADR-0010): the wrapper contract
  every future npm/PyPI/Cargo/Homebrew/Scoop/Winget/Chocolatey package
  must follow, plus scaffolded `distribution/<ecosystem>/` design notes.
  No wrapper is implemented — design and structure only, per scope.
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
  plugin is killed rather than blocking the CLI forever. See ADR-0008.
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
- `markdownlint` now runs in CI against `docs/**/*.md` and the root-level
  `*.md` files, closing a gap where CONTRIBUTING.md documented this check
  before it actually existed.

### Changed

- `engine.Run` now resolves the template and every selected capability
  before invoking any of them (fail-fast — a bad capability id no
  longer lets an earlier capability's side effects happen first).
- `cli` module now depends on `golang.org/x/term` (raw-mode terminal
  input) — isolated to `cli` only; `core`/`sdk/go`/`templates/*`/
  `plugins/*` remain dependency-free.

### Fixed

- `.github/workflows/release.yml` was missing `templates/node-rest-api`
  from the archive-assembly step — every release cut since the Node
  template shipped would have produced archives where `bootstrap new
  --language node` failed with `TemplateNotFoundError`. No release has
  been cut since this template was added, so nothing shipped broken.
- `docs/architecture/plugin-protocol.md` and `CONTRIBUTING.md` referenced
  a non-existent `sdk.Plugin` interface; corrected to the real
  `sdk.TemplatePlugin`/`sdk.CapabilityPlugin` + `sdk.Serve(yourPlugin)`
  shape.
- `docs/architecture/plugin-protocol.md` claimed a capability plugin's
  `plugin.apply` request includes which other capabilities were
  selected — it doesn't; corrected, and tracked as a real gap in
  `roadmap.md`.

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

[Unreleased]: https://github.com/intruder0007/Cli/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/intruder0007/Cli/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/intruder0007/Cli/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/intruder0007/Cli/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/intruder0007/Cli/releases/tag/v0.1.0
