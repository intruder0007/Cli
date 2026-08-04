# Lumo CLI — Design System

> The component library behind every Lumo screen. This is the *how* —
> the vocabulary of reusable parts — while `docs/cli/design-spec.md` is
> the *what* (identity, behavior, rules) and `docs/cli/screen-spec.md`
> is the *where* (each screen as a wireframe). Every command output is
> a composition of the components below; no screen may be hand-rolled.
>
> Scope note: the *Stable command surface* (ADR-0013) is frozen at v1.
> Components are internal presentation; they may change freely until
> the v1 freeze, provided behavior contracts (exit codes, stdout
> determinism, both-theme rendering) hold.

---

## 1. Principles (what modern CLIs taught us, distilled)

Researched Claude Code, Codex, OpenCode, Kiro, and the claudefuel
screenwriting canon, then extracted — not imitated:

1. **Persistent state, not busywork.** A CLI that tells you *what it is
   doing right now* and never lies about it earns trust. Lumo's answer
   is a **persistent, step-based progress bar** (§5) — it stays on
   screen, accumulates checked steps, and never pretends.
2. **Compression is respect.** Every line must earn its rent: one fact
   per line, glyphs over labels where conventional (`✔` not
   `[DONE]`), color as priority signal, never decoration.
3. **Position = meaning.** Same thing always in the same place (banner
   top, plan line first, steps below, hints bottom). Users build a
   mental map; Lumo must not redraw it.
4. **No precision theater.** Percentages come from *real completed
   steps*, timing from a real clock, never a "68%" guess.
5. **Determinism.** Components are pure functions of (theme, state,
   data). Off-TTY output is stable plain text; identical runs produce
   identical transcripts; goldens can be diffed.
6. **Calm confidence.** No flash, no blinking, no surprise reflows, no
   exclamation marks. The tool that respects how little attention it
   deserves is the one users keep open.

---

## 2. Design tokens

### 2.1 Typography

| Token | Value |
|---|---|
| Face | monospace (terminal default; never set a font) |
| Weight | normal; **bold** only for the active step label |
| Size | fixed 1:1; no fractional-width tricks |
| Ellipsis | `…` (minimal theme: `...`) when truncating a long label |

### 2.2 Color tokens (default theme)

| Token | Role | Used by |
|---|---|---|
| `Primary` (cyan) | wordmark, active arrow | Header, Menu, ProgressBar active |
| `Accent` (blue) | plan-line prefix, info | SummaryLine, StatusPanel header |
| `Success` (green) | completed, confirmations | ProgressBar done, Success |
| `Failure` (red) | errors | ErrorPanel, failed step |
| `Warn` (amber) | advisories | ErrorPanel hints, notes |
| `Dim` | secondary text | NextSteps, key hints |
| `Muted` (**new**) | pending steps, captions | ProgressBar pending, captions |
| `Border` (gray) | panel frames | StatusPanel, ConfirmCard |
| `Header` (bold) | step titles | WizardScreen title |
| `Info` | values in reports | Report rows |

The **minimal theme** renders all tokens as plain text — no color, no
style — and substitutes ASCII glyphs (§2.4). Every screen must be
complete in both.

### 2.3 Layout tokens

| Token | Value | Used by |
|---|---|---|
| Indent | 2 spaces | nested content, file trees |
| Column pad | `"  "` after glyphs | all lists |
| Blank line | 1 line between organisms | screen composition |
| Panel rule | `┌ ┐ └ ┘ │` (minimal: `+-`) | StatusPanel, ConfirmCard |
| Max width | 80 cols | all layouts; ellipsized labels |

### 2.4 Glyph tokens

| Meaning | Default | Minimal |
|---|---|---|
| cursor / selected | `❯` | `>` |
| checked / selected | `◉` | `[x]` |
| unchecked | `○` | `[ ]` |
| done step | `✔` | `[x]` |
| failed step | `✖` | `[X]` |
| active step | spinner (`⣾` frames) | `- \ / \|` |
| pending step | `·` | `-` |
| arrow (plan line) | `→` | `>` |
| hint arrow | `↑↓→←` | `^ v > <` |

### 2.5 Timing tokens

| Token | Value |
|---|---|
| Spinner cadence | 700ms/frame (default), 1s (minimal, no animation) |
| Slow-phase note | shown when a phase exceeds 2s |
| Total-time note | shown when generation exceeds 5s |
| Progress refresh | on phase transitions only (no busy-spin) |

---

## 3. Component library

### 3.1 The list

| ID | Component | Kind | Purpose |
|---|---|---|---|
| A-01 | `Text` | atom | styled segment (token + string) |
| A-02 | `Glyph` | atom | single theme-aware status char |
| A-03 | `Caption` | atom | muted commentary (never a glyph) |
| A-04 | `Divider` | atom | horizontal rule between organisms |
| M-01 | `Header` | molecule | wordmark + tagline (welcome) or title |
| M-02 | `SummaryLine` | molecule | the plan line (`→ Creating …`) |
| M-03 | `Step` | molecule | one progress row: glyph + label + timing |
| M-04 | `ProgressBar` | molecule | persistent step stack (see §5) |
| M-05 | `Menu` | molecule | filtered arrow-key list (single select) |
| M-06 | `MultiMenu` | molecule | checkbox list (capabilities) |
| M-07 | `Input` | molecule | line input with inline validation |
| M-08 | `HintBar` | molecule | dim key-hints footer |
| M-09 | `StatusPanel` | molecule | bordered info box (report rows) |
| M-10 | `ConfirmCard` | molecule | bordered decision summary + `y/n` |
| M-11 | `NextSteps` | molecule | numbered follow-up list |
| M-12 | `FileTree` | molecule | indented generated-artifact tree |
| M-13 | `Note` | molecule | inline warn/info note |
| O-01 | `Wizard` | organism | S02–S05: Input + Menu + MultiMenu + ConfirmCard |
| O-02 | `Generation` | organism | S06: SummaryLine + ProgressBar |
| O-03 | `Completion` | organism | S07: Success + FileTree + NextSteps |
| O-04 | `Failure` | organism | S09: ErrorPanel anatomy |
| O-05 | `Report` | organism | S13/S14: StatusPanel rows + notes |
| O-06 | `Help` | organism | S11/S12: sections + HintBar |
| O-07 | `Welcome` | organism | S01: Header + first Input |

### 3.2 Composition rules

1. **Every screen is a list of components** — no hand-rolled layouts.
   `screen := Organism{ components… }` and both themes render it.
2. **Molecules may not nest organisms.** Organisms compose molecules
   and atoms; hierarchy is strict (atom → molecule → organism).
3. **One blank line separates organisms** (token §2.3); molecules
   within an organism are separated by 0 or 1 blank lines per its spec.
4. **Glyph ownership:** a glyph belongs to the component that drew it
   and no other code may print it.
5. **No self-closing animations:** every animated component must have
   a static off-TTY form (§5.4) and a frozen final frame on exit.

---

## 4. Component specs

### A-01 `Text`

`Text(token, str)` — a string tagged with a color token (identity in
minimal theme). Building block of every line; lines are `[]Text`.

### A-02 `Glyph`

`Glyph(meaning)` — resolves §2.4 by theme. Never interpolated into
strings; always composed like `Text("  ", Glyph(Done), " ", "main.go")`.

### A-03 `Caption`

Muted commentary that is *not* output data (e.g. the `# minimal`
legends in the repo's wireframes). `Muted` in default theme, plain in
minimal. Never a glyph prefix.

### A-04 `Divider`

Full-width rule; max 80 cols; used only between organisms, never
inside a molecule. Default: `─` repeated (Border); minimal: `-`.

### M-01 `Header`

Two variants:

- **Brand** (Welcome, Help):
  `Lumo — a new project, ready in seconds`
  (`Primary` wordmark + `Dim` tagline).
- **Section** (WizardScreen, Report, Help sections):
  `Header` token title line, no trailing punctuation.

### M-02 `SummaryLine`

One line; the plan line from S06:

```text
→ Creating my-app — backend-service · go · rest-api · git-init
```

- `→` in `Accent` (default) / `>` (minimal).
- Name in default; type / lang / framework in `Dim`; capability list
  in `Muted`.
- Never wraps; truncates with `…`; optional trailing `(dry-run)`
  `Warn` note.

### M-03 `Step`

One progress row — see §5 for the full lifecycle:

```text
✔ generating template go-rest-api  (1.2s)
⣾ applying capability git-init
· applying capability readme
```

- Glyphs: done `✔`/`[x]` (Success), active spinner (Primary), failed
  `✖`/`[X]` (Failure), pending `·`/`-` (Muted).
- Label: active = bold + default; done = default; pending = Muted;
  failed = bold + Failure.
- Timing `(1.2s)` → `Dim`, appended only to done steps, when > 0.
- The active step's spinner animates in place; the row is never
  re-printed.

### M-04 `ProgressBar`

The persistent step stack — see §5 for the full contract.

### M-05 `Menu`

The existing filtered list (S03):

- Cursor `❯` Primary on the selection row; items `Dim` until cursor.
- Filter: typed characters filter by subsequence (live).
- Keys: `↑↓`/`j`/`k` move, `Home`/`End`/`g`/`G`, `esc` clears filter
  then goes back, `q` quits only when the filter is empty (implemented
  in `cli/internal/prompt/menu.go`).
- Off-TTY: renders the list once, default selection, no animation.

### M-06 `MultiMenu`

S04 capabilities: rows toggle `◉`/`○` (`[x]`/`[ ]` minimal); `space`
toggles; `enter` confirms; `q` quits. Dependencies shown as `Note`
under the group ("requires: …", `Warn`).

### M-07 `Input`

S02 project name: label line + prompt. Renders `Project name: _` with
the cursor as a `❯` block caret interactively, `_` statically off-TTY.
Validation events produce an inline advisory `Note`.

### M-08 `HintBar`

`Dim`, single line, bottom of interactive screens:

```text
↑↓ move · enter pick · type to filter · esc back · q quit
```

Hidden off-TTY and on narrow (< 40 col) terminals.

### M-09 `StatusPanel`

Bordered box (default `┌…┐│…│└…┘`, minimal `+-|`); optional `Accent`
header row; rows are `Text` pairs (`label: value`, label `Dim`, value
default). Used by S08 (target resolution), S14 (doctor checks), S05
(summary).

### M-10 `ConfirmCard`

Bordered panel with the full creation plan (name, target dir, theme,
type, language, framework, capabilities) as `label: value` rows, a
`y/n` prompt line, and a `HintBar`. Used by S05 (new).

### M-11 `NextSteps`

Numbered follow-up list, `Dim`:

```text
1. cd my-app
2. go mod tidy
```

S07 always lists `cd <target>` first (design-spec §6.3).

### M-12 `FileTree`

Indented tree of generated artifacts; directories in `Muted` with a
trailing `/`; files in default; notes in `Note`. Never truncated;
`… (N more)` `Dim` line if the tree exceeds 40 entries.

### M-13 `Note`

Inline advisory: `Note:` label in `Warn` + text. Used for dry-run,
slow-phase timing, and capability dependency notes.

### O-01 `Wizard`

S02→S05 sequence: `Input` (name) → `Menu` (theme) → `Menu` (type) →
`Menu` (language) → `Menu` (framework) → `MultiMenu` (capabilities)
→ `ConfirmCard` (y/n). Each screen = `Header` + component + `HintBar`.
`ConfirmCard` is the last-chance point (esc returns to capabilities;
per design-spec §4.3).

### O-02 `Generation`

S06: `SummaryLine` then `ProgressBar`. Renders immediately on start;
rows appear for every phase as it begins; never cleared.

### O-03 `Completion`

S07: `Success` summary block, capability grouping, `FileTree`,
`NextSteps`. No interactive controls.

### O-04 `Failure`

S09: `[FAIL]` summary (`Failure`), indented `Note` hint (`Warn`), and a
`doctor` escape-hatch line for target/plugin errors. Exit codes per
design-spec §7.4.

### O-05 `Report`

S13 (`plugins list`), S14 (`doctor`): a `StatusPanel` per section plus
`Note` rows. `doctor` renders one row per check
(`✔`/`✖` + text + `Note`).

### O-06 `Help`

S11/S12: `Header` (brand), sections separated by `Divider`, `HintBar`
where interactive. `new -h` (S12) documents the new flags in a
`StatusPanel` of flag defaults.

### O-07 `Welcome`

S01: brand `Header` + `Input` (name) — the banner + first very prompt.
Title is set to `Lumo — new project` on entry (§6).

---

## 5. The persistent step-based progress bar

The core new component: replaces the transient spinner line for
generation (S06) with a **stack of steps that persists and fills**.

### 5.1 State machine

```text
pending → active → done | failed
```

- **pending**: `·` Muted, no timing.
- **active**: `Primary` spinner animated in place; row text fixed.
- **done**: `✔` Success + optional `(1.2s)` `Dim` timing.
- **failed**: `✖` Failure, bold label; the bar freezes with the error
  screen (O-04) below it.

### 5.2 Step definition

Steps come from the engine, never guessed:

1. `generating template` (always first, always one).
2. one `applying capability <name>` per selected capability, in the
   declared order (the engine's `Progress` callback already emits
   these phases — `engine.Progress`).

Total = `1 + len(capabilities)`. The percentage bar is derived purely
from completed steps (`done/total`), refreshed on transitions only.

### 5.3 Anatomy

```text
→ Creating my-app — backend-service · go · rest-api · git-init
✔ generating template go-rest-api                    (0.9s)
✔ applying capability git-init                       (1.1s)
⣾ applying capability readme
· applying capability scaffold-tree
[██████████████░░░░░░░░] 60%
```

- Steps appear when they begin (the row is reserved first).
- Rows are never overwritten; only the active row's spinner advances.
- Percentage bar: `[█…░] N%` (minimal: `[##########----------] 60%`).
- Total time appended after `done`: `completed in 6.2s` (only when
  > 5s, per §2.5).

### 5.4 Off-TTY / piped form

When stdout is not a terminal, the same steps print once, statically,
in order — no spinners, no percentage bar:

```text
→ Creating my-app — backend-service · go · rest-api
✔ generating template go-rest-api
✔ applying capability git-init
✔ applying capability readme
✔ applying capability scaffold-tree
```

Deterministic: identical input → identical transcript (golden-safe).
Each step prints its arrow line as the phase begins (so a hung phase is
visible mid-run), then the line is overtyped in place with the done
glyph (and timing, when the phase took ≥ 0.1s) on completion — a bare
`\r`, never an escape code, so piped bytes stay plain. A failed phase
freezes with the failure glyph; pending rows never print off-TTY.

### 5.5 Slow-phase behavior

If a phase exceeds 2s, the active row's live timing appears
(`⣾ applying capability git-init (2.4s)`) and the `Note`
`Note: this phase is slower than usual — hanging?` prints once, at the
3s mark. Never clears; never guesses.

---

## 6. Terminal title

Every interactive organism (Welcome, Wizard, Generation, Completion,
Failure, Report with live refresh) sets the terminal title to Lumo:

| Scope | Title |
|---|---|
| Entering wizard | `Lumo — new project` |
| During generation | `Lumo — generating my-app` |
| Completion | `Lumo — my-app created` |
| Error | `Lumo — error` |
| All other commands | `Lumo` |

### 6.1 Mechanism

- **POSIX / modern Windows terminals:** OSC 2 sequence
  `\x1b]2;Lumo — new project\x07` written to stdout *only when* stdout
  is a terminal. Errors ignored.
- **Windows legacy console:** `SetConsoleTitleW` via `kernel32`
  (stdlib `syscall` — no new dependency); the previous title is
  captured with `GetConsoleTitleW` so restore is exact.
- **Off-TTY:** nothing is written; output stays deterministic.
- **Restore:** `exit()` in `main.go` restores the previous title on
  every exit path (including `^C` → 130), and `main`'s `defer` covers
  the success path.
- `TERM=dumb` or `NO_COLOR` set: the title still applies (it is not
  color), but no spinner animation.

Implementation: `cli/internal/prompt/title.go` (TTY gate),
`title_windows.go` (kernel32), `title_posix.go` (OSC 2).

---

## 7. Command surface map

Every command composes documented components (principle 6):

| Command | Composition |
|---|---|
| `lumo` (no args) | O-07 Welcome → O-01 Wizard → O-02 Generation → O-03 Completion |
| `lumo new <…> --answers` | O-02 Generation → O-03 Completion (no Wizard) |
| `lumo new <path>` | M-09 StatusPanel (target) → O-02 → O-03 |
| `lumo new -h` | O-06 Help (S12) |
| `lumo plugins list` | O-05 Report (S13) |
| `lumo doctor` | O-05 Report (S14) |
| `lumo --help` / `help` | O-06 Help (S11) |
| any failure | O-04 Failure (S09); the frozen bar stays above it |
| `^C` | cancel + title restore |

---

## 8. Accessibility & determinism contracts

1. **Text-equivalent for every state:** no state exists only as color
   or glyph; the minimal theme proves it (§2.4).
2. **Constrast:** all default-theme color pairs meet WCAG AA on the
   default background (asserted at v0.6.0; see roadmap).
3. **No flashing:** animation is a fixed 700ms spinner alone; nothing
   blinks or pulses; progress updates only on phase transitions.
4. **Keyboard-only:** all interactive components are fully keyboard
   driven (arrows + vim keys); no mouse dependency.
5. **Determinism goldens:** `go test ./cli/...` golden transcripts
   (piped runs, both themes) must be byte-stable; `\r\n`/`\n` are
   normalized only within tests.

---

## 9. Relationship to the other docs

| Doc | Role |
|---|---|
| `design-spec.md` | identity, status line, navigation, errors, exit codes, cross-platform |
| `screen-spec.md` | every screen as wireframes; screens S01–S15 |
| **this file** | the components that compose those screens |
| `design-roadmap.md` | sequencing: which components land in which release |
