# Distribution wrapper protocol

Implements [ADR-0010](adr/0010-distribution-architecture.md). This is
the contract every per-ecosystem distribution wrapper must satisfy,
regardless of what language it's written in. Not to be confused with
[plugin-protocol.md](plugin-protocol.md) (a completely different
contract, between `core` and template/capability plugins).

## The rule

**A wrapper is a thin launcher. It never reimplements the CLI.** It does
not parse `bootstrap`'s flags, render prompts, or know anything about
the plugin protocol. If a wrapper is doing any of those things, it's
wrong — the whole point is that "the exact same CLI experience" holds by
construction, because there is exactly one real implementation (Go) and
every wrapper is a pass-through to it.

## The four steps every wrapper performs

1. **Resolve OS/arch** to the same `<os>_<arch>` pair ADR-0006's release
   matrix uses: `linux_amd64`, `linux_arm64`, `darwin_amd64`,
   `darwin_arm64`, `windows_amd64`. (`windows_arm64` isn't built yet —
   see `roadmap.md`.)
2. **Locate the real binary.** Either:
   - the wrapper's own package already bundled the matching release
     archive (npm/PyPI packages can do this at publish time, one
     platform-specific package variant per target — the standard
     pattern for other CLIs distributed via npm); or
   - download `cli_<version>_<os>_<arch>.{tar.gz,zip}` and
     `SHA256SUMS.txt` from the matching GitHub Release, verify the
     checksum, and cache the extracted archive **as a whole directory,
     siblings intact** — never extract just the `bootstrap`/`bootstrap.exe`
     file on its own, or plugin discovery breaks (see "Why the whole
     archive" below).
3. **Exec, don't shell out through a subshell.** Replace the wrapper
   process with `bootstrap` where the platform supports it (`execve` on
   Unix, `CreateProcess` + wait-and-forward on Windows), or at minimum
   spawn it with stdin/stdout/stderr connected directly (not piped and
   re-emitted) — the interactive wizard's raw-mode arrow-key UI (ADR-0007)
   needs real terminal passthrough; a wrapper that buffers or
   pipes stdio will silently break it.
4. **Forward argv and exit code exactly.** Whatever arguments the user
   passed to the wrapper, pass to `bootstrap` unchanged; exit with
   `bootstrap`'s own exit code.

## Why the whole archive, not just the binary

`cli/main.go`'s `pluginDirs()` looks for `templates/` and
`plugins/builtin/` next to the running executable. A release archive
already ships these as siblings of `bootstrap` (ADR-0006). If a wrapper
extracts only `bootstrap` and discards the rest, `bootstrap plugins
list` finds nothing and `bootstrap new` fails with
`registry.TemplateNotFoundError` — not a wrapper bug exactly, but an
integration mistake this document exists to prevent.

## Version pinning

A wrapper package's own version should track (not necessarily equal,
but traceable to) the `bootstrap` version it launches — e.g. an npm
package `2.0.0` launching `bootstrap v0.1.1` should say so in its
`README`/`CHANGELOG`, so a user filing an issue can tell which `bootstrap`
release they actually hit.

## `go install` — pipeline-built binaries carry the fallback; `go install` itself doesn't yet

`go install github.com/intruder0007/Cli/cli@latest` compiles only the
`cli` binary — no sibling plugin directories exist for it to find, and
this protocol's "locate the real binary" step still doesn't apply
(there's no wrapper in the loop; the binary IS the resolved artifact).
Since ADR-0010: the `cli` binary embeds the V1 plugin set at build time
(`cli/internal/embedded`) and self-extracts it to a version-scoped
cache directory whenever no sibling plugin directories can be found at
all — see
[ADR-0012](adr/0012-universal-install-architecture.md) — so a binary
built by `make build` or the release pipeline works out of the box with
no `CLI_PLUGIN_DIRS` workaround.

The honest caveat: the embedded assets are **staged at build time** into
a gitignored directory (`cli/internal/embedded/assets/`), and only the
Makefile/release pipeline do that staging. `go install` therefore still
produces a binary *without* the embedded fallback, and the `cli`
submodule needs `cli/vX.Y.Z` tags before `@latest` resolves at all —
see [`distribution/go/README.md`](../../distribution/go/README.md) for
the current status.

## Status

Wrappers are implemented and CI-verified for 7 of 8 planned ecosystems
(npm, PyPI, Cargo, Homebrew, Scoop, Winget, Chocolatey) — see
`distribution/<ecosystem>/README.md` for each one's status. **None are
published** to a live registry; publishing is a separate,
explicitly-confirmed step per ecosystem. The Go path (`go install`) is
handled for pipeline-built binaries by the embedded fallback
(ADR-0012); `go install` itself awaits submodule tags and build-time
staging (see the section above). This contract is a Stable public API —
see `docs/architecture/api-compatibility.md`.
