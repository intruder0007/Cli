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

- **Registry-driven wizard (ADR-0007)**: the project type, language,
  framework, and capability menus are built from the installed plugins
  (`core/registry` discovery → `prompt.WizardSpec`), replacing the
  hardcoded option lists. Each step filters the next (language by
  project type, framework by project type + language), so a combination
  with no matching template can't be picked in the first place; the
  capabilities step is skipped when none are installed; and with no
  template plugins discoverable at all, the wizard fails fast with a
  hint instead of asking dead-end questions. Adding a new stack is
  "drop in a template plugin" (ADR-0002) — no CLI changes.
- Type-ahead **fuzzy search in every wizard menu**: typing any letters
  narrows the list by fuzzy subsequence match (`rbapi` finds "REST API
  (node:http)"), `backspace` edits the filter, `esc` clears it first and
  only cancels on a second press, and a hint line under each menu shows
  the active keys. Filtering is a navigation convenience only — the
  line-based fallback and non-interactive flags stay equivalent
  (ADR-0007).
- **Phase progress during `lumo new`**: a plan line (project + choices),
  then one live spinner per phase on a terminal (braille frames in the
  default theme, ASCII in minimal), degrading to plain `- phase` lines
  under pipes/CI — no escape codes off a terminal. Backed by a new
  `engine.Engine.Progress` callback and `prompt.Spinner` (both additive;
  nil Progress stays silent).
- **Design tokens** on `Theme` (`Primary`/`Accent`/`Warn`/`Border`
  helpers + per-theme spinner frames): the whole CLI now renders through
  semantic colors rather than raw ANSI codes, keeping the
  `NO_COLOR`/minimal-theme contract intact (ADR-0007, theme plugin seam).
- Running `lumo` with no arguments now starts the interactive
  wizard (the same as `lumo new`) instead of printing help and
  exiting — this is what double-clicking the binary on Windows does, so
  the wizard actually opens now. Under piped stdin (scripts) it hits
  the wizard's EOF path and fails with "project name is required"
  (exit 1); `lumo help` still prints the command reference.
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
- `lumo plugins validate <dir>` — pre-release extension check:
  validates a plugin directory's manifest (`Manifest.Validate()`) and
  proves the entrypoint binary spawns and passes the
  `plugin.initialize` identity/protocol cross-check against the
  on-disk manifest (the same fail-fast surface `new` uses, ADR-0008),
  without generating anything. Exit 0 = valid. Backed by new
  `core/registry.LoadPluginDir` and `core/plugin.Host.Validate`
  (additive, Experimental surface per ADR-0013).
- Documentation suite (Phase E): new `docs/README.md` index;
  `docs/guides/api-reference.md` (the four Stable surfaces enumerated
  — wire protocol, `sdk/go` exports, CLI commands, wrapper contract);
  `docs/guides/versioning.md` (release/surface/`protocolVersion`
  rules for consumers); `docs/guides/migration-guide.md` (every
  user-visible breaking change so far, the pre-flight checklist, and
  the format future entries must follow); `docs/guides/tutorials.md`
  (a capability plugin in Go, a template plugin in Go, and a
  hand-written protocol implementation using the golden transcripts
  as fixtures). README and api-compatibility now point at the index
  and the migration guide.
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
- The npm wrapper is published as `bootstrap-cli-dev@0.3.0` — the
  platform's first official distribution channel (ADR-0015), verified
  by a clean-machine install from the registry (`lumo version`,
  `plugins list`). Published under the platform's old name; the
  version-synced `lumo-cli` successor is published as `lumo-cli@1.0.0`
  with the v1.0.0 release, replacing the interim package (deprecated
  2026-08-02).
  Production `package.json` (full metadata, `os`/`cpu`
  restricted to the release matrix), a pre-publish guard
  (`scripts/verify-release.js` — fails `npm publish` unless every
  platform archive and `SHA256SUMS.txt` exist for the package's
  version), an idempotency marker so repeated installs skip
  re-downloading the binary, and an actionable shim error when install
  scripts were skipped. `release.yml` gains a `publish-npm` job (runs
  after the GitHub release, gated on the `NPM_TOKEN` secret) with a
  version-sync check, provenance, and a clean-prefix smoke test;
  `npm-verify-published.yml` verifies the published package on clean
  Linux/Windows runners. Release runbook in `docs/guides/releasing.md`.

### Changed

- The wizard no longer shows hardcoded "(coming soon)" placeholders
  (web-app, cli-tool, library, typescript, python, rust, grpc, graphql):
  menus now reflect what's actually installed, and frameworks are
  filtered by the chosen project type + language (previously every
  framework was offered for every language, with unmatched
  combinations failing only at the end). See the migration guide.
- Wizard menus show their position (`Step 1 of 5 · Theme`, …); the
  `lumo new` success screen uses a tree listing (├─/└─ glyphs, `+` in
  minimal) and arrow-marked next steps; the banner tagline is now
  "Lumo — a new project, ready in seconds"; errors get breathing-room
  spacing. Text labels remain the only signal in the minimal theme.
- A bare ESC no longer swallows the keystroke after it (e.g.
  ESC-then-Enter), fixing a lost-input wart in the raw-mode menus.
- All 7 distribution wrappers repointed at the real `v0.3.0` release
  assets (package versions, download URLs, `extract_dir`, and
  checksums from the published `SHA256SUMS.txt`), re-verified against
  them in `distribution-verify.yml` CI. None are published yet; the
  npm wrapper is now publish-ready (see `docs/guides/releasing.md`).
- Codebase audit (Phase F) fixes: `plugin.Host` now wires plugin
  stderr through a new `Stderr` field, verifies the JSON-RPC response
  id, treats `ok:false` handshakes as errors, bounds the shutdown call
  by `ShutdownTimeout`, and logs finish failures; the engine collapses
  duplicate capability selections; `lumo new` refuses non-empty
  target directories and extra positional arguments (exit 2),
  `--answers` + positional project name is rejected (exit 2),
  `config set theme` surfaces config-load errors, `--no-color` now
  also disables theme icons, the line-based wizard cancels cleanly on
  Ctrl+C (exit 130, matching the TUI) and shows capability
  descriptions, and `--answers` files are validated up front. The
  templates return deterministic `FilesWritten`; `github-actions-ci`
  refuses non-Go projects; `git-init` errors name the failing git
  step. CI now runs `make build`. Docs corrected where they
  overclaimed the `go install` path, the non-existent
  `--non-interactive` flag, exit codes, and `go get` of the SDK.
  See `docs/architecture/codebase-audit.md` for the full finding-by-
  finding record.
- `plugins validate -h` usage text expanded (what the check proves,
  exit-code contract).

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
  the bare-binary gap ADR-0010 had left open for binaries built by the
  release pipeline (`make build`/release.yml stage the plugin assets
  into the embed dir first). Note: a raw `go install
  github.com/intruder0007/Lumo/cli@latest` still produces a binary
  without the embedded fallback (the staged assets are gitignored), and
  the module has no `cli/vX.Y.Z` submodule tags yet — see
  `distribution/go/README.md` for the honest status.
- `install.sh`/`install.ps1` — one-line install scripts that resolve
  platform, verify the release archive's checksum, and put just the
  `lumo` binary on `PATH`.
- `lumo doctor` now reports whether this binary has embedded
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
- `lumo new --verbose`/`-v` — prints diagnostic logging (plugin
  spawn, handshake result, timing, file counts) to stderr as a run
  progresses.
- `lumo doctor` — local health check: verifies plugin directories
  resolve and every discovered manifest is valid, with a pass/fail
  summary and recovery hint.
- `lumo version` now prints the Go runtime version and OS/arch
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
  manifest. `lumo plugins list` now reports skipped/invalid
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
- Theme persistence: `lumo config get|set theme`; the interactive
  wizard now remembers your last-chosen theme.
- Redesigned success/error screens; error screen includes a recovery
  hint for a few known failure shapes.
- Startup banner and richer per-command help (`lumo help`,
  `lumo <command> -h`).
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

### Deprecated

- `bootstrap-cli-dev@0.3.0` — the interim npm package published under
  the platform's old name (ADR-0015) — is deprecated on the registry
  (2026-08-02) with a pointer to its successor; `lumo-cli@1.0.0`
  replaces it with the v1.0.0 release (ADR-0016).

### Fixed

- Distribution metadata corruption repaired: single-letter
  substitutions from an earlier bulk edit — `intruder0007/aumo` repo
  URLs in `distribution/npm/package.json`, 13 corrupted words
  including the `NPM_TOKEN` secret name in `distribution/npm/README.md`
  (plus a stale `cli_v<version>` asset name), "hhin launcher"
  descriptions and `MIh` licenses in the cargo/pypi manifests — and
  stale "`lumo-cli@0.3.0` is published" claims in the distribution
  docs, all corrected to the canonical "Lumo project scaffolding
  platform" prose and `github.com/intruder0007/Lumo` URLs. Final
  corruption sweep over the whole repo: zero matches. Packed-tarball
  smoke verified (exactly the five intended files; the pre-publish
  guard fails correctly against the not-yet-existing v1.0.0 assets;
  postinstall fails loudly and the `--ignore-scripts` shim error is
  actionable). Full record: `docs/architecture/npm-identity-migration.md`
  (ADR-0016).

### Fixed

- `.github/workflows/release.yml` was missing `templates/node-rest-api`
  from the archive-assembly step — every release cut since the Node
  template shipped would have produced archives where `lumo new
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

- `lumo plugins list` showed every plugin twice when running a
  released binary from its own extracted directory (the normal usage) —
  `pluginDirs()` now deduplicates its candidate directories by absolute
  path.

## [0.1.0] - 2026-07-31

Initial release. Interactive/non-interactive CLI wizard (theme, project
type, language, framework, capabilities), generating a Go REST API
backend service (`templates/go-rest-api`) with three capability plugins
(`git-init`, `readme`, `github-actions-ci`), over a subprocess +
line-delimited JSON-RPC 2.0 plugin protocol (`sdk/go`).

[Unreleased]: https://github.com/intruder0007/Lumo/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/intruder0007/Lumo/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/intruder0007/Lumo/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/intruder0007/Lumo/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/intruder0007/Lumo/releases/tag/v0.1.0
