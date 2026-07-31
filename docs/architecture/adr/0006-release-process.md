# ADR-0006: Release process — semver tags, cross-compiled matrix, self-contained per-platform archives

## Status

Accepted

## Context

The CLI currently only runs from a source checkout (`go build ./cli` +
plugin binaries built in place, discovered relative to the repo root or
the running executable — see `cli/main.go`'s `pluginDirs()`). For anyone
outside this repo to actually use Cli, there needs to be a downloadable,
versioned artifact that works standalone, with no separate install step
for plugins.

Two release-packaging shapes were considered:

- **CLI-only binary**, plugins installed/fetched separately. Simpler
  archives, but reintroduces exactly the "remote plugin registry" problem
  ADR-0002/`roadmap.md` explicitly deferred, and would ship a CLI that
  does nothing useful out of the box.
- **Self-contained per-platform archive**: the `bootstrap` binary plus
  the V1 template and capability plugins (each cross-compiled for the
  same target), laid out exactly the way `pluginDirs()` already looks for
  them relative to the executable (`<exeDir>/templates/...`,
  `<exeDir>/plugins/builtin/...`). No code change needed in `core`/`cli`
  — only a packaging step that reproduces the same directory shape as a
  local `make build`.

The self-contained shape was chosen: it keeps V1's "no remote registry"
design intact while still being a real, working download.

Since nothing in the repo uses cgo, every target can be **cross-compiled
from a single `ubuntu-latest` runner** via `GOOS`/`GOARCH`, rather than
needing per-OS runners.

## Decision

- **Versioning**: semver tags `vX.Y.Z` on `main`. `cli/main.go`'s version
  string is injected at build time via `-ldflags -X main.version=...`
  (falls back to `"dev"` for unreleased builds), rather than hand-edited.
- **Targets**: `linux/amd64`, `linux/arm64`, `darwin/amd64`,
  `darwin/arm64`, `windows/amd64`. (`windows/arm64` deferred — low
  expected demand for V1, easy to add to the matrix later.)
- **Archive layout** (identical across platforms, only the exe suffix
  changes): `bootstrap(.exe)`, `templates/go-rest-api/{plugin.json,
  go-rest-api(.exe)}`, `plugins/builtin/{git-init,readme,
  github-actions-ci}/{plugin.json, <name>(.exe)}`, plus `LICENSE` and
  `README.md`. Packaged as `.tar.gz` (Linux/macOS) or `.zip` (Windows),
  named `cli_<version>_<os>_<arch>.<ext>`.
- **Trigger**: pushing a `v*.*.*` tag runs
  `.github/workflows/release.yml`, which builds the full matrix, packages
  each archive, generates a `SHA256SUMS.txt`, and publishes a GitHub
  Release via `gh release create` (no third-party release Action — `gh`
  is already the tool used throughout this repo's automation).
- **Changelog**: `CHANGELOG.md` at the repo root, Keep-a-Changelog format,
  with an `Unreleased` section contributors add to per PR (documented in
  `CONTRIBUTING.md`); the release workflow does not auto-generate it —
  moving `Unreleased` into a version section is a manual step before
  tagging, keeping entries human-curated rather than commit-message noise.

## Consequences

- Release engineering has no new runtime dependency for end users (still
  a single self-contained download) and no new build-time dependency for
  contributors (no goreleaser or similar; the workflow is a plain Go
  build matrix + `gh`).
- Adding a template/capability plugin later means adding it to the
  archive-assembly step in `release.yml` — a small, visible checklist
  item, not a hidden gap (worth revisiting if the plugin count grows
  enough that this should be generated from the local `templates/`
  /`plugins/` directory listing instead of hardcoded).
- `windows/arm64` and any remote plugin distribution remain deferred, per
  `roadmap.md`.
