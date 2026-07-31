# ADR-0007: CLI v2 — arrow-key TUI, theme registry, persisted config

## Status

Accepted

## Context

V1's wizard (`cli/internal/prompt`) was plain numbered-list menus and
line input: type a number or id, press enter. It worked, but didn't
"feel like a professional developer tool" — no keyboard navigation, no
true multi-select (comma-separated ids stood in for it), theme was two
hardcoded booleans re-asked every run, no persisted preferences, a flat
global usage string as the only help.

Real arrow-key/space-select navigation requires reading raw terminal
input (bypassing line-buffering) and parsing ANSI escape sequences. Go's
standard library has no cross-platform raw-mode primitive — cooked-mode
line input is all `bufio`/`os.Stdin` give you, and that gap is
particularly sharp on Windows, which has no POSIX termios equivalent in
stdlib at all. Building this from scratch per-OS would be substantial,
fragile, duplicate work; the alternative is a dependency.

## Decision

**Take on `golang.org/x/term`** (maintained by the Go project itself,
not an arbitrary third party) for `IsTerminal`/`MakeRaw`/`Restore`,
**isolated entirely to the `cli` module** — `core`, `sdk/go`,
`templates/*`, `plugins/*` remain at zero dependencies, preserving
ADR-0001/0002's offline-first, dependency-free intent for everything
except the one place that inherently needs real terminal control.

**TTY-aware dual path.** `prompt.RunWizard` checks
`term.IsTerminal` on both stdin and stdout. If both are real terminals:
raw-mode arrow-key/space-select menus (`menu.go`). Otherwise (piped
input, CI, `tests/integration`, any `--answers`/flag-driven invocation):
the exact same line-based prompts V1 shipped, verbatim, relocated
unchanged into `wizard.go`. This is the backward-compatibility
guarantee — no non-interactive usage changes behavior at all.

**Theme becomes a registry** (`theme.go`): a `Theme` struct (rendering
functions + selection glyphs) added via `RegisterTheme`. `default` and
`minimal` are just the first two registrants, at package `init()`. This
is the concrete, honest shape of "future theme plugins" — an in-process
extension point. A full subprocess/JSON-RPC plugin (ADR-0002's transport)
for something as small as a terminal color scheme would be
over-engineering; that isolation guarantee matters for code that
generates files on disk, not for `Success()`/`Failure()` string
formatting.

**Theme persistence** (`config.go`): `os.UserConfigDir()/cli/config.json`.
Only the interactive wizard path writes it — a `--theme`/`--answers`
invocation never mutates persisted state, so scripted/CI usage stays
side-effect-free. `bootstrap config get|set theme` gives explicit
non-interactive control over the same file.

**No animation.** Progress/success/error screens redraw once per state
change, not on a timer/spinner loop — matches "no unnecessary
animations."

**Error screen presentation only.** `suggestFix()` pattern-matches a
handful of known error strings to add a recovery hint. This is
deliberately shallow — a real structured error taxonomy across the
plugin wire protocol (so plugins themselves can return typed,
suggestion-bearing errors instead of opaque strings) is separate,
larger work, tracked for a future plugin-architecture phase, not
attempted here.

## Consequences

- `cli/go.mod` gains `golang.org/x/term` (plus its own transitive
  `golang.org/x/sys`); `go.work.sum` now exists and must be committed
  alongside `go.mod` changes for reproducible builds. No other module's
  `go.mod` changes.
- `menu.go`'s key-reading and selection logic take an injected
  `io.Reader`/`io.Writer` specifically so they're unit-testable without
  a real TTY (simulated ANSI byte sequences) — the only genuinely
  untestable-without-a-real-terminal part is the thin `MakeRaw`/`Restore`
  wrapper in `wizard.go`.
- Raw mode disables the terminal's own signal generation, so Ctrl+C
  arrives as byte `0x03` (handled in `readKey`) rather than a real
  SIGINT; `wizard.go` additionally installs an `os.Interrupt` handler as
  a second line of defense so an external interrupt still restores the
  terminal rather than leaving it stuck in raw mode.
- `cli/internal/prompt/prompt.go` now holds only domain data (`option`
  lists) and `ParseAnswersFile`; `Renderer` is gone, replaced by
  `Theme`; `RunWizard`'s signature changed from
  `(io.Reader, io.Writer) (Answers, error)` to `(io.Writer) (Answers,
  error)` (it now owns `os.Stdin` access directly, needed for the
  raw-mode/TTY-detection branch) — an internal API change, not a
  public/wire-protocol one.
