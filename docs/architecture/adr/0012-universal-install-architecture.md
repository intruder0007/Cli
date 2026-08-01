# ADR-0012: Universal install architecture — embedded self-extracting plugin fallback

## Status

Accepted

## Context

Three real ways to run `bootstrap` existed before this change: a source
checkout (`make build`), a downloaded release archive (ADR-0006), and
`go install github.com/intruder0007/Cli/cli@latest`. The first two work
because `cli/main.go`'s `pluginDirs()` finds `templates/`/`plugins/builtin/`
as siblings of the binary or the working directory. `go install` produces
only the one binary — no sibling directories exist for it to find — so
`bootstrap new` fails with `TemplateNotFoundError` on the single most
idiomatic install path for a Go CLI tool.

ADR-0010/`distribution-protocol.md` already named this exact gap and its
recommended direction ("`go:embed` self-extraction") but explicitly
deferred it as future work. This ADR is that future work, now done: the
underlying problem is that **plugin availability was coupled to how the
binary was distributed**, instead of being an intrinsic property of the
binary itself.

## Decision

**Embed the V1 plugin set into the `cli` binary via `go:embed`,
self-extracted to a version-scoped cache directory, engaged only as a
last-resort fallback.**

- **Lives in `cli`, not `core`** (`cli/internal/embedded`). `core`'s own
  stated principle is that it never hardcodes what it can generate;
  baking the specific V1 plugin binaries into `core` would violate that
  in spirit even though `go:embed` doesn't create a Go import. Only
  `cli`, the distribution-facing leaf binary, knows which plugins V1
  ships. `core`/`sdk/go` are unchanged — the module-boundary table in
  `overview.md` still holds exactly as written.
- **Strictly a fallback.** `pluginDirs()`'s existing precedence
  (`CLI_PLUGIN_DIRS` override → exe-relative → cwd-relative) is
  unchanged and still wins whenever it finds anything. A cheap probe
  (`hasAnyPlugin`, existence-only, no manifest parsing) checks whether
  any candidate directory actually contains a plugin before the
  fallback ever engages. This is zero-regression by construction — the
  new code path never runs for an existing archive install or dev
  checkout — and avoids re-introducing the exact "plugin listed twice"
  class of bug already fixed once in this project (`pluginDirs`'s
  `dedupeAbs`, `v0.1.1`).
- **Version-scoped cache directory**:
  `os.UserCacheDir()/bootstrap/<version>/`, mirroring
  `cli/internal/prompt/config.go`'s existing use of `os.UserConfigDir()`
  for the same kind of per-user, per-OS standard directory (cache, not
  config, since this is regenerable). Scoping by version means an
  upgrade can never serve stale plugins from a previous version's
  cache. `registry.scanDir` treats a configured directory's *immediate*
  children as plugin directories, so the fallback appends
  `<cacheDir>/templates` and `<cacheDir>/plugins/builtin` as two
  entries — matching the exact shape the existing sibling-directory
  candidates already use — not the bare cache root.
- **Build-order coupling, made explicit.** `go:embed` can only embed
  files already on disk inside `cli`'s own module tree at build time —
  it cannot reach into the sibling `templates/`/`plugins/builtin/`
  modules directly. `Makefile`'s `build` target now depends on a new
  `stage-embedded` target that builds every V1 plugin and copies each
  binary + `plugin.json` into `cli/internal/embedded/assets/` before
  `cli` builds. `release.yml` does the same per target, inside the
  existing per-platform loop, so each platform's `bootstrap` binary
  embeds only that platform's plugin binaries. A checked-in
  `assets/.gitkeep` placeholder keeps a plain `go build ./cli` (without
  the Makefile's staging step) compiling — it just embeds nothing, and
  `embedded.Available()` reports false, making self-extraction a
  harmless no-op for that build.
- **Install script layer** (`install.sh`, `install.ps1`, repo root):
  resolves OS/arch, downloads the matching release archive +
  `SHA256SUMS.txt` from the latest GitHub Release, verifies the
  checksum, and installs *only* the `bootstrap` binary onto PATH
  (`~/.local/bin` by default on Unix, `%LOCALAPPDATA%\Programs\bootstrap`
  on Windows — PATH is never modified silently, only reported).
  Because the binary is now self-sufficient, the script doesn't need to
  preserve sibling directories at the install location at all — a
  direct, practical consequence of this design, not a separate feature.
- **`bootstrap doctor`** gained an "Embedded fallback:" section
  reporting whether this binary has embedded assets at all, and whether
  the fallback is the thing actually serving plugins for the current
  run — so `doctor` stays the single diagnostic surface for "why can't
  it find my plugins," covering this path too.

## Consequences

- `cli/main.go`'s `pluginDirs()` gained new fallback logic
  (`hasAnyPlugin`, `embeddedCacheDir`, `embeddedFallbackDir`) and a new
  import (`cli/internal/embedded`); `cmdDoctor` gained
  `printEmbeddedStatus`. Both are internal, not public/wire-protocol
  changes.
- `Makefile` and `release.yml` both gain a real, new build-order
  dependency: plugins must build (and be staged) before `cli` builds,
  for the same target. This is a meaningful process change for
  contributors and CI, not just an implementation detail — called out
  explicitly here rather than left implicit in a diff.
- The release archive format is **unchanged** — it still ships
  `bootstrap` plus sibling `templates/`/`plugins/builtin/` directories,
  exactly as ADR-0006 defined. Simplifying the archive now that the
  binary is self-sufficient is a deliberate non-decision: no measurable
  benefit yet, real regression risk (every existing distribution doc
  and would-be package-manager formula assumes today's archive shape),
  so it's left alone per "don't break working architecture unless
  there's a measurable benefit." Worth revisiting once the embedded
  fallback has real-world mileage.
- No garbage collection exists for old-version cache directories left
  behind by a prior `bootstrap` version. Not proven to matter yet;
  tracked as tech debt rather than built speculatively.
- Verified end-to-end, not just unit-tested: a `cli` binary built with
  no sibling plugin directories anywhere near it, run from a directory
  with nothing next to it and no `CLI_PLUGIN_DIRS`, successfully runs
  `bootstrap doctor` and `bootstrap new` via the self-extracted cache —
  this is the literal `go install` failure mode, reproduced and proven
  fixed (`tests/integration`'s `TestEndToEndGenerateViaEmbeddedFallback`,
  plus a manual repro during this phase's verification).
