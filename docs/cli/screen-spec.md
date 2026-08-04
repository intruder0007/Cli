# Lumo CLI — Screen Spec

> Companion to `docs/cli/design-spec.md`. Every screen Lumo can render,
> as text-level wireframes, in both themes. Convention: `❯` cursor /
> `◉` checked / `○` unchecked are default-theme glyphs; `>` / `[x]` /
> `[ ]` are minimal-theme equivalents. `#` marks a caption, not
> rendered output. Screens marked **new** are not yet implemented.

---

## S01 — Banner (interactive wizard entry)

```text
# default
Lumo — a new project, ready in seconds

Project name: _

# minimal (identical text)
Lumo — a new project, ready in seconds

Project name: _
```

- Printed only when the wizard starts interactively (`lumo`, `lumo new`
  with no flags/answers).
- The name step follows immediately (see S02).

---

## S02 — Project name step

```text
# default, empty input — placeholder dimmed
Project name: my-app (or a path like ../work/app)_

# default, mid-typing with live advisory validation
Project name: my-ap[p
  name may contain only letters, digits, - or _

# default, valid path input
Project name: ../work/my-app_
```

Behavior:

- Live validation is advisory; enforced on Enter.
- Enter runs `resolveTargetPath` (bare name / relative / absolute / `~`),
  then `Answers.Validate`.
- Invalid on Enter → S09 (error screen) with the name hint.
- `Esc` clears input; `Ctrl+C`/`q` cancels (S10).

---

## S03 — Menu step (type / language / framework / theme)

```text
# default
Step 2 of 5 · Theme

  ❯ default   color + icons
    minimal   plain text, NO_COLOR friendly

  filter: · ↑/↓ or j/k move · enter select · esc back · q quit

# minimal
Step 2 of 5 · Theme

  > default   color + icons
    minimal   plain text, NO_COLOR friendly

  filter: · ↑/↓ or j/k move · enter select · esc back · q quit
```

Filtered state:

```text
Step 4 of 5 · Framework

  ❯ Go REST API Service
    Node.js HTTP API Service      # both match filter "api"

  filter: api (2 of 2) · ↑/↓ move · enter select · esc clear · q quit
```

Zero-match state:

```text
  filter: xyz (0 of 2)
  no matches — edit or clear the filter (esc)
```

- Step numbering `N of M` reflects skipped steps (capabilities absent →
  4 steps).
- Borders/box-drawing only in default theme; minimal renders a plain
  list.

---

## S04 — Capabilities step (multi-select)

```text
# default
Step 6 of 7 · Capabilities

  ❯ ◉ git-init              Initialize Git repository
    ○ readme                Enhance README
    ○ github-actions-ci     GitHub Actions CI (go build + go test)

  filter: · space toggle · enter continue · esc back · q quit
```

- Skipped entirely when no capability plugins are installed.
- `Space` toggles the highlighted row; selection persists across
  back/forward navigation.
- If a capability has `dependsOn`, a dimmed `depends on <id>` note is
  appended; selection that would violate dependency order is blocked
  with a one-line warn (`requires <id>` — select it first).

---

## S05 — Confirm step **new**

```text
# default
Step 7 of 7 · Confirm

  Target:   C:\Users\me\code\my-app
  Stack:    backend-service · go · rest-api
  Extras:   git-init, readme

  enter generate · ← back · q quit

# minimal
Step 7 of 7 · Confirm

  Target:   C:\Users\me\code\my-app
  Stack:    backend-service · go · rest-api
  Extras:   git-init, readme

  enter generate · ← back · q quit
```

- Target is the **resolved absolute path** (the user sees the real
  location; `~`/relative expansions become concrete here).
- `Enter` prints the plan line (S06) and begins generation (S07).
- Always the last screen before disk writes. No further questions.

---

## S06 — Plan line + generation progress

```text
# default (interactive, TTY)
  → Creating my-app — backend-service · go · rest-api · git-init, readme
  ⠋ generating template go-rest-api

# default, one phase done, next in flight
  → Creating my-app — backend-service · go · rest-api · git-init, readme
  ✔ generating template go-rest-api
  ⠙ applying capability git-init

# default, complete
  → Creating my-app — backend-service · go · rest-api · git-init, readme
  ✔ generating template go-rest-api
  ✔ applying capability git-init
  ✔ applying capability readme

# minimal / piped / --no-color (no animation, no escape codes)
  - Creating my-app — backend-service · go · rest-api
  - generating template go-rest-api
  - applying capability git-init
```

- Slow-phase note (additive, only when a phase > 2s):
  `✔ applying capability readme (3.1s)`.
- Spinner occupies only the current line; finished phases become
  permanent lines.
- On failure mid-run: `✗ applying capability readme` then S09.

---

## S07 — Success screen

```text
# default
✔ Generated my-app
  template: go-rest-api · 7 file(s) · capabilities: git-init, readme

  + .gitignore
  + Makefile
  + README.md
  + go.mod
  + internal/http/router.go
  ├─ internal/http/router_test.go
  └─ main.go

Next steps:
  → cd my-app
  → go build ./...
  → go test ./...
  → go run . # serves on :8080, GET /healthz

# minimal
[OK] Generated my-app
  template: go-rest-api · 7 file(s) · capabilities: git-init, readme

  + .gitignore
  + Makefile
  + README.md
  + go.mod
  + internal/http/router.go
  + internal/http/router_test.go
  + main.go

Next steps:
  - cd my-app
  - go build ./...
  - go test ./...
  - go run . # serves on :8080, GET /healthz
```

With capability grouping (additive):

```text
✔ Generated my-app
  template: go-rest-api · 7 file(s) · capabilities: git-init, readme

  template:
  ├─ go.mod
  ├─ main.go
  └─ internal/http/router.go

  capability git-init:
  └─ .git/

  capability readme:
  └─ README.md

Next steps:
  → cd my-app
  → go run .
```

- Slow-run note (only when total > 5s): dimmed `completed in 6.2s` after
  the file list.
- `cd <target>` is always first (CLI-computed from the resolved path).

---

## S08 — Target-path resolution screens (non-interactive)

```text
# bare name from CWD — plan line only
  → Creating my-app — backend-service · go · rest-api

# explicit relative path
  → Creating ../work/my-app — backend-service · go · rest-api

# explicit absolute path
  → Creating C:\code\my-app — backend-service · go · rest-api
```

- Non-interactive runs print the plan line, then progress, then S07/S09.
- The positional argument may be name or path (2026-08-03
  `resolveTargetPath`); `--dir` (new) separates name and location.

---

## S09 — Error screen

```text
# default
[FAIL] project name "2bad" must start with a letter and contain only letters, digits, - or _
  hint: project names must start with a letter and contain only letters, digits, - or _.

# minimal
[FAIL] target directory C:\code\my-app already exists and is not empty
  hint: pick a different name, or generate into an empty/new directory.
```

Patterns (all map to typed errors in the v0.6+ taxonomy):

| Failure | Hint |
|---|---|
| template not found | `lumo plugins list` + `LUMO_PLUGIN_DIRS` |
| capability not found | `lumo plugins list`, exact `capabilityId` match |
| plugin start error | rebuild plugin binary |
| protocol mismatch | rebuild against current `sdk/go` |
| identity mismatch | stale binary vs `plugin.json` |
| plugin timeout | check plugin stderr / `--verbose` |
| capability cycle | inspect `dependsOn` declarations |
| target exists non-empty | choose another name/dir |
| bad project name | name rules |
| no plugins at all | `lumo doctor` (primary escape hatch) |

---

## S10 — Cancelled

```text
# default
cancelled

# minimal
cancelled
```

- Exit `130`. Nothing written to disk.

---

## S11 — Usage / help

```text
usage: lumo <command> [flags]

Running lumo with no arguments starts the interactive wizard
(the same as 'lumo new').

commands:
  new [project-name-or-path]   generate a new project (interactive
                               wizard if no flags/answers given; run
                               'lumo new -h' for non-interactive flags)
  plugins list                 list discovered template and capability
                               plugins
  plugins validate <dir>       check a plugin directory before shipping
  config get theme             print the persisted theme
  config set theme <name>      persist a theme (default|minimal)
  doctor                       run local health checks
  version                      print the CLI version and platform
  help                         this text

Run 'lumo <command> -h' for flags on a specific command.
Pass -verbose (or -v) to 'lumo new' for diagnostic logging on stderr.
```

- Unknown command: prints help to **stderr**, exit `2`.

---

## S12 — `lumo new -h` **new**

```text
usage: lumo new [project-name-or-path] [flags]

Generates a new project. With no flags and no --answers, runs the
interactive wizard. Flags below make it non-interactive.

The positional argument may be a bare project name (generated in the
current directory) or a target path — relative (./my-app, ../x/app),
absolute (/home/me/app, C:\code\app), or ~-prefixed (~/code/app). The
project name is derived from the path's final component.

flags:
  --dir <path>            generate into <path>, name from positional
  --theme <name>          CLI theme: default or minimal
  --project-type <t>      project type, e.g. backend-service
  --language <l>          language, e.g. go
  --framework <f>         framework, e.g. rest-api
  --capabilities <csv>    comma-separated capability ids
  --answers <file>        path to an answers file (name may be a path)
  --no-color              disable color output
  --verbose, -v           print diagnostic logging to stderr
```

---

## S13 — `lumo plugins list`

```text
go-rest-api       template    0.1.0    Go REST API Service
node-rest-api     template    0.1.0    Node.js HTTP API Service
git-init          capability  0.1.0    Initialize Git repository
github-actions-ci capability  0.1.0    GitHub Actions CI (go build + go test)
readme            capability  0.1.0    Enhance README
```

- Skipped-but-invalid plugin directories print to stderr
  (`skipped <path>: <cause>`), then the valid list.
- No plugins: `no plugins found` (exit 0 — listing is a query, not a
  health check).

---

## S14 — `lumo doctor`

```text
Plugin directories:
  C:\Users\me\code\lumo\templates
  C:\Users\me\code\lumo\plugins\builtin
  C:\Users\me\AppData\Local\lumo\v0.4.0\templates   (embedded fallback)

Embedded fallback:
  plugin assets are embedded in this binary
  in use: no sibling plugin directories were found, so plugins were
  self-extracted to C:\Users\me\AppData\Local\lumo\v0.4.0

Plugins:
  go-rest-api (template) v0.1.0
  git-init (capability) v0.1.0
  ...

doctor: all checks passed
```

- Any failed check → `doctor: found issues` + a hint, exit `1`.
- New in the design: doctor reports **which plugin set served the last
  run** (embedded vs sibling) and the count of embedded assets — the
  "why did my binary generate / not generate" answer at a glance.

---

## S15 — Slow-phase and verbose diagnostics (`--verbose`)

```text
  → Creating my-app — backend-service · go · rest-api
  ⠙ applying capability readme (3.1s)
  verbose: resolved template "go-rest-api" for backend-service/go/rest-api
  verbose: run complete: 9 files written, 2 capabilities applied
```

- Verbose lines go to **stderr** (existing), prefixed `verbose:`, never
  to stdout (stdout stays machine-parseable).
- Timing lines (see §5.2 of the design spec) appear in normal output
  only when a phase exceeds the threshold.

---

## Screen inventory

| ID | Screen | Theme variants | Status |
|---|---|---|---|
| S01 | Banner | both | implemented |
| S02 | Project name step | both | implemented (path support new) |
| S03 | Menu step | both | implemented |
| S04 | Capabilities step | both | implemented (dependsOn note new) |
| S05 | Confirm step | both | **new** |
| S06 | Plan + progress | both | implemented (persistent ProgressBar new) |
| S07 | Success | both | implemented (grouping new) |
| S08 | Target resolution (non-interactive) | both | implemented |
| S09 | Error | both | implemented (typed taxonomy new) |
| S10 | Cancelled | both | implemented |
| S11 | Help | both | implemented |
| S12 | `new -h` | both | **new** |
| S13 | `plugins list` | both | implemented |
| S14 | `doctor` | both | implemented (serving-source note new) |
| S15 | Verbose diagnostics | both | implemented |

---

## Component map

> Every screen above is a composition of the components defined in
> `docs/cli/design-system.md`. `[kbd]` marks the fully keyboard-driven
> components; `[static]` marks render-once components with no live
> redraw. S06 is rendered by the persistent `ProgressBar` (M-04) —
> `cli/internal/prompt/progress.go` — the transient spinner's
> replacement (design-system §5).

| Screen | Components used |
|---|---|
| S01 Banner | O-07 Welcome → A-02 Glyph, M-01 Header (brand), M-07 Input |
| S02 Project name | M-07 Input, M-13 Note (advisory), M-08 HintBar |
| S03 Menu | M-05 Menu `[kbd]`, M-08 HintBar |
| S04 Capabilities | M-06 MultiMenu `[kbd]`, M-13 Note (depends-on), M-08 HintBar |
| S05 Confirm | M-10 ConfirmCard, M-08 HintBar |
| S06 Plan + progress | M-02 SummaryLine, M-04 ProgressBar (M-03 Step rows), M-13 Note (slow) |
| S07 Success | M-02 SummaryLine, M-12 FileTree, M-11 NextSteps |
| S08 Target resolution | M-09 StatusPanel, M-13 Note |
| S09 Error | O-04 Failure: M-13 Note (hint), M-08 escape hatch |
| S10 Cancelled | M-13 Note (Warn) |
| S11 Help | O-06 Help: M-01 Header, A-04 Divider, M-08 HintBar |
| S12 `new -h` | O-06 Help + M-09 StatusPanel (flag defaults) |
| S13 `plugins list` | O-05 Report: M-09 StatusPanel, M-13 Note |
| S14 `doctor` | O-05 Report: M-09 StatusPanel (check rows), M-13 Note |
| S15 Verbose | A-01 Text (stderr), M-13 Note (timing) |
