# Codebase audit (Phase F)

A written audit of the whole `cil` codebase, its tests, its CI, and its
documentation, conducted before the v0.4.0 release. Findings are listed
with their fixes; anything left open is explicitly marked as residual
and tracked in [`roadmap.md`](roadmap.md).

## Scope and method

- **Audited surfaces:** `cli/` (command dispatch, wizard, themes,
  embedded fallback), `core/` (`config`, `diag`, `engine`, `plugin`,
  `registry`), `sdk/go/`, the four V1 plugins
  (`templates/go-rest-api`, `templates/node-rest-api`,
  `plugins/builtin/{git-init,readme,github-actions-ci}`),
  `tests/integration`, `.github/workflows/`, `distribution/`, and the
  docs tree.
- **Method:** two parallel read-only passes (one over cli/core/tests/CI,
  one over templates/plugins/distribution/docs), then targeted
  verification of every finding against the source — including running
  the binary paths the docs describe (e.g. trying the documented
  `go install` claims against the actual module/tag layout), and
  cross-checking doc claims against `cli/main.go`'s real flag set.

## Findings and fixes

### Core (`core/`)

| ID | Finding | Fix |
|---|---|---|
| F16 | `plugin.Host` had no way to receive a plugin process's stderr — it was always discarded, hiding a hung or failing plugin's own output even under `--verbose`. | New `Host.Stderr io.Writer` field; `start()` wires it to the subprocess. `bootstrap new --verbose` now sets it to `os.Stderr` (`core/plugin/host.go`). |
| F17 | `call()` did not verify the response's JSON-RPC `id` — a stale line from an earlier timed-out call could be attributed to a new request. | Response id must equal the request id, else an explicit error (`core/plugin/host.go`). |
| F18 | `initialize()` treated a response with `ok:false` as success and proceeded to compare manifests. | `ok:false` is now a hard error (`core/plugin/host.go`). |
| F19 | `finish()` gave the shutdown call a full `CallTimeout` budget even though the plugin already failed a call. | The shutdown call is bounded by `ShutdownTimeout` (`core/plugin/host.go`). |
| F20 | `finish()` swallowed every failure (shutdown-call error, kill-on-hang) silently. | Failures are logged via the host logger; the kill path is now observable (`core/plugin/host.go`). |
| F21 | `resolveOrderedCapabilities` resolved and ran duplicate capability ids (e.g. `--capabilities git-init,git-init` ran git-init twice against the same target). | Selection is deduplicated (first occurrence wins) before resolution (`core/engine/engine.go`). |
| F22/F27 | `sortByDependencies` indexed by capability id; a duplicate id would collide. | Resolved by F21's dedupe — each id now appears exactly once. |

### CLI surface (`cli/`)

| ID | Finding | Fix |
|---|---|---|
| F23 | `bootstrap new` generated into an existing directory unconditionally — plugins write files without checking, so re-runs silently overwrote a previous project. | A target that exists and is non-empty (or is a file) is refused up front with a clear error, before any plugin runs (`cli/main.go`). |
| F24 | Ctrl+C on the wizard's project-name prompt (raw mode delivers it as byte 3) produced a confusing "project name is required" instead of cancelling. | `readLineRaw` reports the cancel; the wizard maps it to `ErrCancelled` (exit 130). The raw-mode SIGINT handler is now registered before `term.MakeRaw`, closing the window where an interrupt couldn't restore the terminal (`cli/internal/prompt/wizard.go`). |
| F25 | The line-based wizard (non-TTY fallback) had no Ctrl+C handling — the process just died mid-prompt. | A SIGINT handler prints `cancelled` and exits 130, matching the TUI path (`cli/internal/prompt/wizard.go`). |
| F26 | `config set theme` ignored `LoadConfig` errors (`cfg, _ :=`), so a corrupted config file was silently overwritten. | The error is surfaced (`cli/main.go`). |
| F28 | `--answers <file>` combined with a positional project name silently ignored the name (two sources of truth). | Refused up front, exit 2 (`cli/main.go`). |
| F29 | Extra positional arguments to `new` were silently ignored. | Refused up front, exit 2, naming the offending argument (`cli/main.go`). |
| F30 | `--no-color`/`NO_COLOR` disabled color but left the theme's glyph icons (`❯`, `◉`, `○`, `✔`, `✗`) — the wrong half of "plain output". | `GetTheme` now forces `UseIcons` off along with `UseColor` (`cli/internal/prompt/theme.go`). |
| F31 | The line-based wizard's capability step printed bare ids (`1) git-init`) with no descriptions, unlike the TUI. | The list now shows name and id, and names are accepted as answers (`cli/internal/prompt/wizard.go`). |
| F32 | `plugins validate -h` printed only the usage line, no description of what the check does. | Full usage text including the exit-code contract (`cli/main.go`). |
| F33 | An `--answers` file that omitted required fields failed only deep in the engine, after plugin discovery. | `ParseAnswersFile` validates the whole answer set immediately, naming the file in the error (`cli/internal/prompt/prompt.go`). |

### Plugins and templates

| ID | Finding | Fix |
|---|---|---|
| F34 | Both templates returned `FilesWritten` in map-iteration order — nondeterministic across runs/OSes, and the engine can compare it against other plugins' lists. | Keys are iterated sorted (`templates/{go-rest-api,node-rest-api}/main.go`). |
| F35 | `github-actions-ci` wrote a Go workflow (`setup-go`, `go build`) into any project regardless of language — a Node project would get a CI workflow that can never be green. | The capability now refuses non-Go languages with a clear error before writing anything (`plugins/builtin/github-actions-ci/main.go`). |
| F36 | `git-init` errors didn't say which git step failed, and claimed `.git/` ambiguously. | Errors are wrapped per step (`git init:` / `git add:` / `git commit:`); the `FilesWritten` semantics (the created `.git/` repository directory, not the template's source files) are documented in the code (`plugins/builtin/git-init/main.go`). |

### Documentation

| ID | Finding | Fix |
|---|---|---|
| F37 | README and the v0.3.0 changelog claimed `go install .../cli@latest` "works out of the box". In reality the embedded assets are staged at build time into a gitignored directory, so a `go install`-built binary has no plugin fallback — and the `cli` submodule has no `cli/vX.Y.Z` tags, so `@latest` doesn't even resolve. | README's install block, the changelog entry, and `distribution/go/README.md` now state the honest status (binary yes; embedded fallback only via the release pipeline; tags needed). |
| F38/F39 | `distribution-protocol.md`'s "go install — mitigated" section and `roadmap.md` repeated the same overclaim. | Both corrected; the wrapper contract's Status section now distinguishes pipeline-built binaries from a raw `go install`. |
| F40 | `api-reference.md` documented a `--non-interactive` flag that doesn't exist (non-interactivity is implicit when a positional name or any flag is given). | The contract row now says exactly that. |
| F41 | `usage.md` documented `plugins validate`'s exit codes but nothing else; several paths (usage errors, cancels) had undocumented exits. | A full **Exit codes** section (0/1/2/130) was added to `docs/cli/usage.md`. |
| F42 | Both tutorials told readers to `go get github.com/intruder0007/Cli/sdk/go@latest`; no `sdk/go/vX.Y.Z` tags exist, so the command fails today. | The `replace`-directive and in-repo alternatives are documented at both spots (`docs/guides/tutorials.md`). |

### CI

| ID | Finding | Fix |
|---|---|---|
| F44 | CI built and tested every module but never ran `make build` — the staging-then-embed pipeline that every shipped binary depends on (ADR-0012's build-order coupling) could break without CI noticing. | A "Build via Makefile (staged embedded pipeline)" step runs `make build` when `go.work` is present, mirroring the existing module guard (`.github/workflows/ci.yml`). |

## Verified as correct (no fix needed)

- **Embedded-cache isolation:** `CLI_PLUGIN_DIRS` always wins over the
  embedded fallback (`TestPluginDirsOverrideSkipsFallback`), and the
  cache is version-scoped (`TestEmbeddedCacheDirIsVersionScoped`).
- **Wire-protocol goldens:** byte-exact on both sides of the wire;
  regenerated only via `-update`, kept LF via `.gitattributes`.
- **Manifest validation:** `Manifest.Validate()` covers every required
  field per kind; registry tests cover valid/missing/invalid manifests.
- **Version injection:** consistent via ldflags from `git describe`
  (`Makefile`, `cli/main.go`).
- **Docs index and lint:** `markdownlint-cli` clean across the repo
  after the fixes.

## Residual items (tracked, not fixed)

- **`go install` path:** needs `cli/vX.Y.Z` (and `sdk/go/vX.Y.Z`)
  submodule tags, and a decision on build-time staging — documented in
  `distribution/go/README.md`, `roadmap.md`, and this report.
- **Old-version embedded-cache cleanup:** already tracked in
  `roadmap.md`; not proven to matter yet.
- **Capability conflict detection:** ordering exists (ADR-0008),
  conflict detection doesn't; already tracked in `roadmap.md`.
- **`make build` on Linux** leaves extensionless built binaries in the
  plugin source directories; `.gitignore` covers `*.exe` (Windows) but
  not the extensionless Linux builds. Harmless (gitignored regions are
  documented in the Makefile), worth a one-line gitignore addition
  later.

## New tests added by this audit

- `core/config/config_test.go` — `Answers.Validate` table (valid,
  defaults, bad names, unknown theme, missing fields).
- `core/engine/engine_test.go` — duplicate capability selections are
  collapsed in both resolution and apply order.
- `core/plugin/host_test.go` — `initialize` with `ok:false`; stale
  response-id rejection; matching response-id acceptance; `finish()`
  kills a hung process after `ShutdownTimeout`; `finish()` lets a
  cooperative process exit cleanly; `Host.Stderr` reaches the plugin
  process (all via a scripted `TestHelperProcess` subprocess or a
  built helper binary).
- `tests/integration/integration_test.go` — extra positional args
  (exit 2), `--answers` + positional (exit 2), non-empty target
  directory (exit 1, existing file untouched), unknown `--theme`
  (exit 1), `config set theme` with an unknown theme (exit 1).
