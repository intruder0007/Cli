# ADR-0010: Distribution architecture — per-ecosystem wrappers, design only

## Status

Accepted

## Context

**Naming collision, resolved first.** This project already has an "SDK"
— `sdk/go`, the library *plugin authors* use to implement the JSON-RPC
protocol (ADR-0002). Phase 4 of the mandate asks for a different thing
entirely: a way for *end users* on different package-manager ecosystems
(npm, PyPI, Cargo, Homebrew, Scoop, Winget, Chocolatey, plus Go's own)
to install and run the `bootstrap` CLI naturally within their existing
tooling. Both get called "SDK" colloquially, but they don't share code,
a protocol, or a purpose. To avoid this becoming permanently confusing
for contributors, this document calls the new concept **distribution
wrappers**, never "SDK," and `sdk/go` keeps sole claim to that name.

**What already exists to build on**: ADR-0006 already produces, per
release, a self-contained per-platform archive
(`cli_<version>_<os>_<arch>.{tar.gz,zip}`) containing `bootstrap` plus
every template/capability plugin as sibling directories, plus
`SHA256SUMS.txt`. `cli/main.go`'s `pluginDirs()` already resolves
plugins relative to the running executable's own directory. This means
**most distribution channels need zero new runtime code** — only a thin
per-ecosystem wrapper that fetches/unpacks the existing archive (as a
unit, siblings intact) and execs the real binary.

**The one real gap**: `go install .../cli@latest` compiles *only* the
`cli` binary — no sibling `templates/`/`plugins/` directories exist for
`pluginDirs()` to find. This is a genuine architectural question, not
solved by the "reuse the archive" insight above, and is called out
explicitly rather than glossed over.

Per the mandate: **design only**. No wrapper is implemented in this
phase — interfaces, folder structure, and the abstraction contract are.

## Decision

1. **Distribution wrappers are thin launchers, never reimplementations.**
   Every wrapper — regardless of language — does exactly four things:
   resolve the current OS/arch, locate or fetch the matching release
   archive (verified against `SHA256SUMS.txt`), exec the real
   `bootstrap` binary with argv/stdin/stdout/stderr passed through
   *unmodified*, and exit with its exact exit code. None of them parse
   flags, render prompts, or touch the plugin protocol — that guarantees
   "the exact same CLI experience" *by construction*: there is only ever
   one real implementation of the CLI (Go), everything else is a
   pass-through. The full contract is
   [docs/architecture/distribution-protocol.md](../distribution-protocol.md).
2. **Archive-based channels (Homebrew, Scoop, Winget, Chocolatey, and
   any npm/PyPI wrapper that downloads-on-install) install the archive
   as a unit** — the formula/manifest must not cherry-pick just the
   `bootstrap` binary out of the extracted directory, or plugin
   discovery breaks. This is now a documented constraint, not an
   implicit assumption.
3. **`go install`'s gap is documented, not solved here.** Recommended
   future direction: embed the plugin binaries into the `cli` binary via
   `go:embed` and self-extract to a cache directory on first run — but
   that's real implementation work for a future phase, tracked in
   `roadmap.md`, not decided irreversibly by this ADR.
4. **Folder structure**: `distribution/<ecosystem>/README.md` for each
   of npm, pypi, cargo, go, homebrew, scoop, winget, chocolatey — each a
   short design note (status, what building it actually requires, link
   to the spec), not a working wrapper. `distribution/README.md` is the
   index with a status table.

## Consequences

- No new runtime code, no new tests — this phase is entirely
  documentation and empty-but-real directory structure.
- The next contributor who wants to ship an npm wrapper (for example)
  has a precise, unambiguous contract to implement against, and doesn't
  need to reverse-engineer it from the release workflow.
- The `go install` gap is now a tracked, named problem instead of a
  silent one — worth resolving before recommending `go install` as a
  primary install path in user-facing docs.
