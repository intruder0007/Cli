# ADR-0011: Developer experience polish — logging seam, doctor, richer version/summary output

## Status

Accepted

## Context

Phase 5 of the mandate asks for improved logging, diagnostics,
validation, helpful errors, recovery suggestions, command help, project
summaries, generated-project information, and version output — every
interaction should feel polished. Prior phases already delivered typed
errors and recovery hints (ADR-0008) and a project-summary-shaped
`engine.Summary`; what was still missing was *visibility* into what the
engine and plugin host are actually doing during a run, a way for a
contributor or user to self-diagnose a broken local setup without
reading source, and version output thin enough to be useless when
triaging an environment-specific bug report.

## Decision

- **`core/diag` — a minimal logging seam, not a logging framework**: a
  single-method `Logger` interface (`Logf(format string, args
  ...interface{})`), a `NoopLogger` (default, zero-cost, zero-alloc-ish),
  and a `WriterLogger` (timestamped lines to any `io.Writer`). `core`
  stays free of any opinion about terminals, color, or verbosity levels —
  it only ever calls `Logf`. `cli` decides whether/where output goes.
- **Nil-safe injection, not a required dependency**: `core/plugin.Host`
  and `core/engine.Engine` each gained an exported `Logger diag.Logger`
  field plus an unexported `logger()` helper that returns `NoopLogger{}`
  when the field is nil. Every existing caller (including all of
  `tests/integration`) is unaffected — logging is opt-in, never assumed.
- **Log at lifecycle boundaries, not line-by-line**: spawn, handshake
  result, generate/apply success or failure with timing and file counts,
  capability resolution and execution order, run completion. Enough to
  answer "what did the CLI actually do and how long did each step take"
  during a `--verbose` run, without turning into wire-protocol-level
  tracing (which belongs to a future structured-tracing effort, if ever
  needed — not attempted here).
- **`bootstrap new --verbose` / `-v`**: constructs a `diag.WriterLogger{W:
  os.Stderr}` and assigns it to both the `Host` and `Engine`. Stderr, not
  stdout, so it never interleaves with the wizard/success-screen output
  a script might parse or a user might read top-to-bottom.
- **`bootstrap doctor`**: a new subcommand that runs the same
  `Registry.DiscoverWithIssues()` fail-fast surface `plugins list`
  already uses, plus a check that each configured plugin directory
  exists, and prints a pass/fail summary with a recovery hint on
  failure. Deliberately spawns no plugin subprocess — it's a *discovery*
  health check, not a full dry-run; a full generate dry-run is out of
  scope for this increment (see Future recommendations).
- **Richer `version` output**: `bootstrap version` now prints the Go
  runtime version and `GOOS`/`GOARCH` alongside the semver string
  (`bootstrap version v0.2.0 (go1.22.x, windows/amd64)`), matching what a
  bug report needs without requiring a separate `go env` dump from the
  reporter.
- **Project summary on success**: `engine.Summary` gained `Template` and
  `CapabilitiesApplied` fields (populated by `Engine.Run`, which already
  had this information); `prompt.SuccessScreen`'s signature changed from
  five loose parameters to `(w, t, projectName, s engine.Summary)` so it
  can render a one-line summary (`template: go-rest-api · 8 file(s) ·
  capabilities: git-init, readme`) before the file list.

## Consequences

- `prompt.SuccessScreen`'s signature is a breaking change within the
  `cli` module (its only caller, `cli/main.go`, was updated in the same
  change) — not a public/wire-protocol break, since neither `sdk/go` nor
  the plugin protocol changed.
- `core/plugin.Host` and `core/engine.Engine` each gained one exported
  field; existing zero-value construction (`plugin.NewHost()`,
  `engine.New(...)`) is unaffected since both default to a no-op logger.
- `bootstrap doctor`'s scope is intentionally narrow (discovery only).
  If a future need arises to validate that a *specific* template or
  capability actually generates successfully end-to-end without leaving
  side effects, that likely wants a `--dry-run` on `bootstrap new`
  itself (writing to a temp dir and discarding it) rather than growing
  `doctor` into a second code path — noted in the roadmap, not built
  here.
