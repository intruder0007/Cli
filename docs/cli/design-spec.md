# Lumo CLI — Design Specification

> Principal-architect design, 2026-08-03. Defines the **target** user
> experience for Lumo as a world-class open source developer product.
> Companion documents: `docs/cli/screen-spec.md` (wireframes for every
> screen) and `docs/cli/design-roadmap.md` (phased delivery to v1.0.0).
> This spec is normative for design decisions; it does not override the
> stability contract in `docs/guides/versioning.md` (ADR-0013) — the
> command/flag surface is Stable at v1, so changes must land before the
> v1 freeze or ship as additive, opt-in additions.

---

## 1. Identity

### 1.1 What Lumo is

Lumo is a **project scaffolder with a strong opinion about momentum**:
the time between "I have an idea" and "there is a compiling project with
tests, a README, and CI" should be measured in seconds, not tutorials.
Everything in the UX follows from that single promise.

Three identity pillars, in order of precedence:

1. **Velocity without noise.** Lumo moves fast and says little. Default
   output is a plan line, a progress line, and a result. Nothing more.
   Every additional line of output must justify its existence.
2. **A hand on the shoulder, not a wall.** When something fails, Lumo
   tells you *what happened, why, and what to do next* — in that order —
   instead of a stack trace. Failure is a conversation, not a verdict.
3. **Deterministic and honest.** The same command produces the same
   output. Progress is never faked. "Coming soon" is never printed as if
   it were here. No telemetry, no ads, no updates nagging.

### 1.2 Voice

- **Tone:** calm, direct, first-person-plural-free. Lumo speaks in the
  imperative for actions ("Run `cd my-app`"), in plain declaratives for
  state ("3 files written").
- **No cheerleading.** "Great job!", emoji celebrations, and ASCII art
  are banned. The single banner line is the entire brand statement.
- **No pathos.** Errors do not apologize; they diagnose.
- **Vocabulary:** "generate", "apply", "discover", "target". Never
  "deploy", "ship", "scaffold-out" — Lumo's verbs are the engine's verbs
  (`core/engine`, `core/registry`), so the mental model of the UI is the
  mental model of the architecture.

### 1.3 The banner

```text
Lumo — a new project, ready in seconds
```

- Printed **only** before the interactive wizard, never for commands.
- One line, no ASCII art, works in both themes.
- Rationale: the wizard is a *place* the user enters; commands are
  *tools* the user invokes. The banner marks the boundary.

### 1.4 Version identity

`lumo version` prints `lumo version vX.Y.Z (go1.26.x, windows/amd64)`.
This is the canonical bug-report string. Never prints "dev" in a
distributed build: the release pipeline injects the real tag via
`-ldflags "-X main.version=vX.Y.Z"` (ADR-0006), and the Makefile's
`stage-embedded` + `git describe` path does the same for local builds.

---

## 2. The status line

### 2.1 Definition

The **status line** is a single structural line that frames one unit of
work. There are exactly two variants, and every screen in the CLI uses
one of them:

| Variant | When | Shape |
|---|---|---|
| **Plan line** | Before a run (`lumo new`) | `→ Creating <target> — <type> · <lang> · <framework> [· capabilities]` |
| **Section header** | Inside wizard / doctor / list output | `Step 2 of 5 · Theme` (wizard), `Plugin directories:` (doctor) |

The plan line is the *contract* of a run: it states the target, the
stack, and any capabilities — before anything happens. If the run fails,
the plan line is still there, so the user knows what *was* attempted.

### 2.2 Rules

1. **One status line per screen.** Never two competing headers.
2. **The plan line is printed to stdout, before progress begins.**
3. **The plan line degrades gracefully**: with `--no-color`/minimal
   theme the `→` becomes `-` (existing behavior, kept).
4. **Never contains secrets or absolute home paths** unless the user
   passed them explicitly (a target path the user typed is fine; the
   derived absolute location is not printed if it differs).

---

## 3. Navigation model

### 3.1 Screens, not pages

Lumo uses a **single-focus screen model**:

- One question (or one status) per screen, nothing else on it.
- The screen is **re-rendered atomically**: on input, the entire screen
  is redrawn (clear + redraw), never appended-to. This is what the
  wizard's raw-mode path does today via `menu.go`; the spec makes it a
  rule for every interactive screen.
- A screen is defined by: `Header (status line)` + `Body (one
  question/state)` + `Footer (key hints)`.

### 3.2 Keys — universal, learnable once

| Action | Primary | Alternative |
|---|---|---|
| Move up / down | `↑` / `↓` | `k` / `j` |
| Confirm / select | `Enter` | — |
| Multi-select toggle | `Space` | — |
| Filter (type-to-find) | any alphanumeric | — |
| Clear filter | `Esc` | — |
| Back to previous screen | `←` | `h` |
| Cancel (exit wizard) | `Ctrl+C` | `q` (when no filter text) |
| Jump to first/last | `Home` / `End` | `g` / `G` |

Rules:

1. **`q` never conflicts with filtering.** If the filter buffer is
   non-empty, `q` types into the filter; only an empty filter + `q`
   cancels. (Existing behavior — codified here.)
2. **`Esc` clears the filter first**, then (second `Esc`) acts as
   "back". One key, two contexts, deterministic.
3. **Vim keys only apply to menus**, never to text input.
4. Every menu footer prints the keys that apply *to that screen* — the
   help is contextual, not global.

### 3.3 Back and cancel

- **Back (`←`/`h`)** returns to the previous screen with the previous
  answer intact. Capabilities already toggled are preserved.
- **Cancel (`Ctrl+C`/`q`)** aborts the whole wizard: prints `cancelled`,
  exits `130` (SIGINT convention), writes **nothing** to disk.
- After the plan line is printed (generation has begun), `Ctrl+C` is the
  plugin protocol's job (timeouts + process teardown, `core/plugin`);
  Lumo surfaces what was written before the interruption.

### 3.4 The wizard flow (target)

```text
name → theme → type → language → framework → capabilities → confirm → generate
```

| Step | Widget | Notes |
|---|---|---|
| 1. Project name | text input | Accepts a bare name **or a target path** (relative/absolute/`~`) — implemented 2026-08-03 (`resolveTargetPath`). |
| 2. Theme | menu | Persisted (ADR-0007). |
| 3. Project type | menu | From discovery (`WizardSpec`). |
| 4. Language | menu | Filtered by type. |
| 5. Framework | menu | Filtered by type + language. |
| 6. Capabilities | multi-select | Skipped if none installed. |
| 7. Confirm | summary card | **New in the design**: a read-only recap with "generate" / "back" / "cancel". |

Step 7 exists to close the loop: the user *sees* the exact plan line
that will be printed, and confirms it. It costs one Enter and converts
"surprise generation" into "intentional generation" — the single most
trust-building screen in the wizard.

### 3.5 Filtering

Type-to-filter is **subsequence match** (existing): `rbapi` finds
"REST API (node:http)". Rules:

- Filter matches against display names and ids.
- The filtered list shows a `filter: <text> · N of M` indicator.
- Backspace edits; Esc clears; arrow keys still navigate the filtered
  list; Enter confirms the highlighted row.
- A filter with zero matches shows the list empty, the indicator
  `no matches`, and Enter is disabled (must clear or edit).

---

## 4. Wizard flow details

### 4.1 Step structure (every step is identical)

```text
╭─ Step 2 of 5 · Theme ────────────────────────────────╮
│                                                      │
│  ❯ default   color + icons                           │
│    minimal   plain text, NO_COLOR friendly           │
│                                                      │
│  filter: · ↑/↓ move · enter select · esc back · q quit │
╰──────────────────────────────────────────────────────╯
```

1. Header: `Step N of M · <name>` (M reflects skipped steps — e.g. 4
   when no capability plugins exist).
2. Body: the widget.
3. Footer: contextual keys, always one line, dimmed.
4. Borders drawn only when the theme uses icons; minimal theme renders a
   plain list without box-drawing.

### 4.2 The name step

```text
Project name: my-app
```

- Placeholder text dimmed inside the input when empty: `my-app (or a
  path like ../work/app)`.
- Live validation: while typing, an invalid character (space, leading
  digit, `/` inside what would be a name — but `\`, `/` are allowed when
  the text is a path) shows a dimmed hint under the input. Validation is
  **advisory while typing, enforced on Enter.**
- On Enter: full resolution via `resolveTargetPath` + `Answers.Validate`
  with the existing error screen on failure.

### 4.3 The confirm step (new)

```text
Step 7 of 7 · Confirm

  Target:   C:\Users\me\code\my-app
  Stack:    backend-service · go · rest-api
  Extras:   git-init, readme

  Press Enter to generate · ← back · q quit
```

- Shows the **resolved absolute target** (the one real path the run will
  touch), the stack, and capabilities.
- `Enter` = generate; `←` = back to capabilities; `q` = cancel.
- No theme question here; theme was step 2.

---

## 5. Generation progress

### 5.1 States

Generation renders one line per phase (existing `engine.Progress`
contract), in order:

1. `generating template <name>`
2. `applying capability <id>` (× N, in dependency order)

### 5.2 Rendering

- **Interactive (TTY):** the current phase label is drawn inline with
  the theme's spinner frames (existing `spinner.go`); each completed
  phase resolves to a final line `✔ generating template go-rest-api`
  (or `-` in minimal). The previous phase's finished line is kept; the
  spinner occupies the current line only.
- **Piped/CI:** no spinner, no escape codes; each phase prints a plain
  line `- generating template go-rest-api` as it begins (existing
  degradation, kept).
- **Timing (new, additive):** a finished phase may append `(0.4s)` when
  a phase exceeds 2 seconds — a dimmed, honest signal that this phase is
  slow. Never shown for fast phases; never fake.

### 5.3 Rules

1. Progress is driven by real phase callbacks, never a timer.
2. The final line of progress is the first line of the completion
   screen — no blank gap, no double newline.
3. On failure mid-run, the last started phase is marked `✗` (or `[FAIL]`)
   and the error screen follows — the user sees exactly where it broke.

---

## 6. Completion screens

### 6.1 Success

```text
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
```

Rules:

1. **File tree is the file list.** The list is indented under the
   summary, sorted as the plugin reports them, with the existing
   `├─/└─` glyphs in the default theme and `+` in minimal. Files
   written by capabilities are visually grouped after the template's
   files (two dimmed group headers: `template:` / `capability:`).
2. **Next steps come from the plugins** (`Summary.NextSteps`), never
   hardcoded by the CLI.
3. `cd <target>` is always the first next step; the CLI prepends it
   from the resolved target (it knows the real path, the plugin doesn't).
4. **Nothing else.** No "you're all set!", no ASCII art, no time taken
   (unless > 5s, then one dimmed line `completed in 6.2s`).

### 6.2 Cancelled

```text
cancelled
```

- Single line, info-styled, exit `130`.

### 6.3 What Lumo never prints

- Advertisement of other Lumo features at completion ("try `lumo config`!")
- Emoji, sparkles, celebration
- The full absolute target when it equals the display path

---

## 7. Errors

### 7.1 The error screen

```text
[FAIL] target directory C:\code\my-app already exists and is not empty
  hint: pick a different name, or generate into an empty/new directory.

```

Anatomy (in order):

1. **The failure line** — what failed, in one sentence, no prefix
   redundant with the command ("generating from template ..." wraps the
   real cause via `%w`; the real cause is what's shown).
2. **The hint line** — one actionable fix, dimmed, prefixed `hint:`.
   Pattern-matched today (`suggestFix`); **new**: a typed, structured
   error taxonomy (see §7.3).
3. **Nothing else.** Exit code carries the severity.

### 7.2 Exit codes (Stable at v1)

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Runtime failure (validation, resolution, plugin, filesystem) |
| `2` | Usage error (unknown command/flag, bad combination) |
| `130` | Cancelled (Ctrl+C / `q`) |

These are part of the Stable surface (ADR-0013) — scripts depend on
them, so they are enumerated in the v1 semver guarantee.

### 7.3 Structured error taxonomy (new, additive)

`core/errors` gains a small typed hierarchy (all `%w`-compatible):

- `ValidationError{Field, Value, Hint}` — bad name, bad theme, missing
  field.
- `ResolutionError{What, ID, Hint}` — template/capability not found.
- `PluginError{Name, Phase, Cause, Hint}` — spawn, handshake, timeout,
  identity mismatch, generation/apply failure.
- `TargetError{Path, Reason, Hint}` — exists-and-non-empty, is-a-file,
  unreadable.

Each carries a `Hint()`; `suggestFix` becomes a thin dispatcher over
`errors.As`, and `--verbose` prints the full chain. **No error string
changes** (Stable surface: error *text* is not Stable, but churn costs
tests/goldens, so the taxonomy maps 1:1 to today's strings).

### 7.4 `lumo doctor` as the error escape hatch

Every error screen whose hint ends in "see `lumo doctor`" — or that
mentions discovery (`no template plugins found`, `plugins list`) — is
reproducible as `lumo doctor` output. Doctor is the health-check
command, and it is the single answer to "why is my Lumo broken".

---

## 8. Accessibility

### 8.1 Color and icons are additive, never sole signals

Already true (ADR-0007, `usage.md`): every state has a text label
(`[OK]`/`✔`, `[FAIL]`/`✗`). Maintained as a hard rule: any new status
introduced by the spec (e.g. the slow-phase timing note) must have a
text form.

### 8.2 NO_COLOR and --no-color

- `NO_COLOR` env var and `--no-color` force color+icons off (existing).
- `--no-color` wins over a persisted theme (existing).
- New: `--no-color` also disables the spinner animation (renders plain
  phase lines) — animation is a color-adjacent distraction; `NO_COLOR`
  is often set by users who also want no motion.

### 8.3 Screen readers and plain terminals

- The **minimal theme** (`minimal`) is the screen-reader/plain path:
  no box-drawing, no unicode glyphs, `>` cursor, `[x]` checkboxes,
  ASCII spinner, single-space columns.
- Wizard works **without raw mode** via the piped/line fallback (existing
  ADR-0007 dual path) — any automation or assistive setup can drive it
  line-by-line.

### 8.4 Contrast and motion

- Color choices (cyan/blue/amber on default background) verified for
  WCAG AA (4.5:1) against both light and dark terminals — a lint-time
  check in the theme registry (new, additive): `RegisterTheme` asserts
  defined token contrast ratios.
- **No flashing.** No animations beyond the spinner's rotation; no
  blinking cursor tricks; screens redraw atomically (never accumulate).

### 8.5 Keyboard-only operation

Every interactive action has a keyboard path (section 3.2); there is no
mouse-only feature. Focus is never "trapped" in the wizard: `q`/`Ctrl+C`
always exit.

---

## 9. Performance

### 9.1 Budgets

| Phase | Budget |
|---|---|
| Cold start to first output (`lumo new --help`, `version`) | < 50 ms |
| Discovery (5 V1 plugins, embedded fallback, warm cache) | < 100 ms |
| Wizard screen redraw | < 16 ms |
| First phase line after Enter on confirm | < 200 ms |
| Full V1 generation (Go template + 3 capabilities) | < 5 s cold |

Budgets are measured by the existing `--verbose` timing lines; a
`lumo doctor`-style perf check is **not** added — timing is a
diagnostic (`--verbose`), not a feature.

### 9.2 Rules

1. **Lazy discovery.** Discovery runs once per process, only when a
   screen needs it; the wizard's per-step filtering reuses the one
   `WizardSpec` (existing).
2. **Embedded fallback is cheap.** `embedded.ExtractTo` is idempotent
   and skips an existing, non-empty version-scoped cache dir (existing,
   ADR-0012) — a warm cache costs one `os.ReadDir`.
3. **No network, ever.** Lumo does not phone home, check for updates,
   or download anything at runtime (distribution downloads happen at
   install time, checksum-verified).
4. **Memory bound.** A run must never hold more than ~2× the generated
   project in RAM; file writes stream.
5. **Parallel plugin spawn (new, v0.6+).** Template generation and
   capabilities are sequential today (dependency order exists and must
   stay); where no dependency exists between capabilities, they may be
   prepared in parallel — but *writing* stays ordered, so output is
   deterministic. Default: sequential; `--verbose` shows the chosen
   order.

---

## 10. Theming

### 10.1 The theme registry (existing, kept)

`Theme` is data (ADR-0007): name, `UseColor`, `UseIcons`, cursor and
checkbox glyphs, spinner frames. `RegisterTheme` is the seam. The spec
freezes this contract as the v1 theming surface.

### 10.2 Design tokens

Tokens are the semantic layer `Theme` renders from (existing helpers):
`Primary` (brand cyan), `Accent` (blue), `Warn` (amber), `Border`
(gray), `Success` (green), `Failure` (red), `Dim`, `Header`, `Info`.

New token (additive): **`Muted`** for the per-screen footer key hints —
`Dim` is overused today (hints, group headers, tree glyphs all Dim);
`Muted` gives the footer a consistent, distinguishable quietness.

### 10.3 Themes at v1

| Theme | Color | Icons | For |
|---|---|---|---|
| `default` | yes | yes | everyday use |
| `minimal` | no | no | plain terminals, screen readers, scripts |

- Two themes is the v1 number. More shipped themes = more contrast
  testing = more surface; the *registry* is the extensibility story,
  and shipping 2 is honest.
- **`LUMO_THEME` env var** (new, additive): overrides the persisted
  theme for one invocation, `--theme` flag overrides both. Precedence:
  `--theme` > `LUMO_THEME` > persisted config > `default`. (`NO_COLOR`/
  `--no-color` still strip color/icon from whatever wins.)
- **Theme validation:** `config set theme` accepts only registered
  names (existing `ThemeNames()`); unknown names in config fall back to
  `default` with a `Warn` line once (today: silent fallback).

### 10.4 Theme plugins stay in-process

ADR-0007's decision stands: a color scheme does not need the plugin
protocol's isolation. A future `lumo themes` command (post-v1) installs
into `os.UserConfigDir()/lumo/themes/` as data files parsed at startup —
still in-process, still the same `Theme` struct.

---

## 11. Cross-platform behavior

### 11.1 The platform matrix

| Concern | Behavior |
|---|---|
| Paths | `filepath` throughout; `\`/`/` both accepted on Windows; `~` expands (2026-08-03); absolute/relative/bare targets identical on all OSes. |
| Executables | Entrypoint resolution appends `.exe` on Windows (`resolveEntrypoint`); PATHEXT never assumed for explicit paths. |
| Raw mode | `golang.org/x/term` (ADR-0007); Windows console VT support; falls back to line mode when not a TTY. |
| Signals | `Ctrl+C` → exit 130 with cleanup; Windows console handler parity via the same raw-mode wrapper. |
| Line endings | Generated files use LF (`.gitattributes` pins goldens); the CLI writes exactly what plugins write — no line-ending rewriting. |
| Cache/config dirs | `os.UserCacheDir()/lumo/<version>` (ADR-0012), `os.UserConfigDir()/lumo/config.json` — correct per-OS locations, no `~/.lumo` hardcoding in code (installer's job, v0.5.0). |
| Unicode | Icons/spinners only when the theme enables them and the terminal is a TTY; `minimal` guarantees pure-ASCII output. |
| Colons in paths | Windows drive letters are handled by `filepath.Abs`/`Base`/`Dir` — never hand-split path strings. |

### 11.2 The double-click experience (Windows)

Double-clicking `lumo.exe` opens the wizard (existing `len(os.Args) < 2`
→ `cmdNew(nil)`). Spec addition: when stdin is **not** a TTY (console
window from Explorer counts as a TTY; a piped/detached context does
not), and the run fails fast, Lumo prints the error **and pauses** with
`Press Enter to close` — a console window that flashes and vanishes is
an accessibility failure. This is the final piece of the "lumo.exe
appears to do nothing" class of bugs (fixed 2026-08-03: staged embedded
assets + killed stale `bootstrap.exe` process).

### 11.3 Determinism

- Same command + same plugins + same input = byte-identical output on
  every OS (golden transcripts exist for the wire protocol; extend the
  discipline to CLI output screens in v0.6.0 tests).
- Capability order is dependency-resolved and stable (Kahn's algorithm,
  `engine.go`).
- No timestamp, PID, or absolute-path leakage into generated files
  unless a template deliberately includes one.

---

## 12. Command surface (v1 freeze)

The Stable command/flag surface (ADR-0013), unchanged from today except
the additions marked **new**:

| Command | Stable flags / forms |
|---|---|
| `lumo` / `lumo new` | positional name-or-path (2026-08-03), `--theme`, `--project-type`, `--language`, `--framework`, `--capabilities`, `--answers`, `--no-color`, `--verbose`/`-v`, `--dir` **new** |
| `lumo plugins list` | — |
| `lumo plugins validate <dir>` | — |
| `lumo config get/set theme` | `default`/`minimal` |
| `lumo doctor` | — |
| `lumo version` | — |
| `lumo help` | — |

**`--dir <path>` (new, additive):** explicit target-directory flag for
scripts that want the name and location separate (`lumo new my-app
--dir ../work`). Resolves through the same `resolveTargetPath` rules;
mutually exclusive with a positional path (error, exit 2). This gives
scripting the full expressiveness of the wizard without new parsing.

---

## 13. Design rules of thumb (for reviewers)

1. If a screen has more than one status line, it's wrong.
2. If output is not deterministic for the same input, it's wrong.
3. If a failure doesn't end with a hint, it's incomplete.
4. If an animation conveys information that text doesn't, it's wrong.
5. If a new feature needs a new key that isn't in §3.2, reconsider.
6. If the minimal theme can't render a screen, the screen isn't done.
