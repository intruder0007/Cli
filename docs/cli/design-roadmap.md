# Lumo CLI — Design Roadmap to v1.0.0

> Companion to `docs/cli/design-spec.md` and `docs/cli/screen-spec.md`.
> This plan slots the design work into the existing release sequence in
> `docs/architecture/v1-readiness-report.md` (v0.4.0 → v0.8.0 → v1.0.0)
> **without breaking the semver contract** (ADR-0013): the Stable
> command/flag surface is frozen at v1, so every design change below
> either ships before the v1 freeze or is additive.

Principle: **the CLI surface and its UX behavior are part of what v1
stabilizes.** The design is not a "post-v1 refresh" (that was the old
stance) — it is the *shape of the v1 experience*, sequenced so that v1
credentials are not made stale a month after the release.

---

## Design phases mapped to releases

```text
v0.4.0   v0.5.0         v0.6.0          v0.7.0          v0.8.0 (RC)     v1.0.0
(stops   (DX:           (struct:        (platform:      (everything     (freeze +
the      install,       errors,         --dir,          stable,         docs
drift)   the startup   taxonomy,       confirm step,   DRY-RUN,        truth)
         experience)    redraw, a11y)   deps notes)     soak)
```

### Already delivered (v0.4.0 window, 2026-08-03)

- **Root-caused the "lumo.exe appears dead" bug.** The shipped
  `bootstrap.exe` was a *staged* build (embedded V1 plugin set); the
  user's `lumo.exe` was a *plain* `go build ./cli` (embedded only the
  `.gitkeep` placeholder) — so it could not discover plugins and exited
  immediately. Fixed by rebuilding with `make build`'s staging, killing
  a lingering `bootstrap.exe` process, and documenting the trap in the
  build docs (§9/§14 of `ai-context-report.md`).
- **Path-aware `new`.** `lumo new ./a/b/app`, `lumo new /abs/path/app`,
  `lumo new ~/code/app` and the bare-name form all generate correctly;
  the project name is the path's final component
  (`cli/main.go: resolveTargetPath`), with regression tests.
- **Answers-file paths.** `projectName:` in an answers file may now be a
  target path (`config.Answers.ValidateShape` defers the name-pattern
  check to resolution).

### v0.5.0 — The startup experience (design: part 1)

Theme: **installation and first run feel inevitable.**

1. **`--dir` flag** (design spec §12). Separate name and location for
   scripts: `lumo new my-app --dir ../work`. Same resolver, no new
   parsing. Additive.
2. **`lumo new -h` screen (S12).** Today `new` has no per-command help
   beyond the flag defaults printed by `flag`; replace with the
   documented screen. Additive.
3. **Installer work (from the v1-readiness report) carries the UX:**
   first-run banner + `doctor` hint if the binary has no embedded
   assets. A binary without staging must *say so* at first use, not
   flash a console.
4. **Slow-phase timing note** (§5.2) and `--no-color` disables
   animation (§8.2). Behavior-only, no API change.

Exit: `lumo new -h` is complete; `--dir` works for name+location
scripts; a mistakenly unstaged binary explains itself.

### S2 — Lands in the v0.5.0 window — Structural errors

Theme: **failure is a conversation.**

1. **`core/errors` typed taxonomy** (§7.3): `ValidationError`,
   `ResolutionError`, `PluginError`, `TargetError` — each with `Hint()`,
   all `%w`-compatible. `suggestFix` becomes a dispatcher; `--verbose`
   prints the chain. **No error-text churn** (tests/goldens stay green).
2. **Error screen discipline** (§7.1): every failure line is the real
   cause (through `%w` wrapping), every screen ends in a hint or a
   pointer to `lumo doctor`. Completion screen (S07) capability grouping.
3. **`doctor` serving-source note (S14)** — answer "which plugin set
   served me?" at a glance.

Exit: `new`/`plugins`/`doctor` paths all route failures through the
taxonomy; doctor says which plugin source served the run.

### S3 — Lands in the v0.6.0 window — Polish, accessibility, redraw

Theme: **the inside as clean as the outside.**

1. **Atomic redraw everywhere** (§3.1). Audit every interactive screen
   for redraw-on-input / no-append (spinner/progress already canonical).
2. **Accessibility pass (§8):** contrast-ratio assertion in
   `RegisterTheme`; `Muted` token; text labels for every status;
   minimal-theme purity guide. `NO_COLOR`/`--no-color` parity cases
   tested (as part of v0.6.0's negative-test expansion).
3. **Determinism backlog** (§11.3): extend golden-transcript discipline
   to CLI *screens* where a screen is exercised in tests.
4. Markdownlint over the new spec docs (this tree).

Exit: every screen redraws atomically; a11y contract enforced in tests;
new docs lint-green.

### S4 — Lands in the v0.7.0 window — The determination of a product (UX depth)

Theme: **the wizard is where Lumo makes its pitch.**

1. **Confirm step (S05)** — the trust screen: resolved absolute target,
   stack, extras, Enter to generate. This is the biggest single new
   note in the interactive flow (no accidental writes). Additive to the
   wizard (defaults to "Enter to confirm"; so `Ctrl+C` still aborts, no
   step adds friction on the happy path because Enter was already the
   last keypress).
2. **Capabilities dependencies UX (S04)** — `depends on <id>` dimmed
   note and a one-line guard when selecting would violate an order.
3. **`--dir` wired into Confirm screen** display.

Exit: a first-time user can do a full wizard run with no "fill" hidden;
errors are always recoverable; the surface reset.

### S5 — Lands in the v0.8.0 window (RC) — Soak and finish

Theme: **freeze and prove.**

1. **Descriptive usage/error freeze** — re-read the spec for any section
   still *not* delivered; ship the remainder or explicitly defer with a
   dated ADR note.
2. **Roadmap linkage closed.** `design-roadmap.md` — checked at cut
   time against actual mark.
3. **Cross-platform matrix via `doctor -v`** (existing `doctor`, add
   `--verbose` to also print active platform-specific discovery roots,
   cache paths, embedded status — the "which binary am I running, really"
   answer for bug reports).

### Design system build-out (component library — cross-cutting, S1–S5)

The component library in `docs/cli/design-system.md` lands incrementally
so no release window is forced to ship a big-bang rewrite. UI text is
destination-agnostic: the component output is specified first, then the
screen is "recomposed" — not rewritten.

| Window | Components to land | Code milestone |
|---|---|---|
| v0.4/v0.5.0 | `TerminalTitle` (§6), `Muted` token, `Step` passthrough | title set/restore in `main.go`; tokens in `theme.go` |
| v0.5.0 | M-03 `Step`, M-04 `ProgressBar` (persistent, S06), M-02 `SummaryLine` reuse | `prompt` progress stack replacing transient spinner |
| v0.6.0 | M-01..M-08 refactor of `menu.go`/`prompt.go` to component renderers; both-theme parity | all interactive components reuse `Text`/`Glyph` |
| v0.7.0 | M-09/M-10 `StatusPanel`/`ConfirmCard` (S05, S08, S14) | confirm screen + doctor/plugins report panels |
| v0.8.0 | O-03..O-07 organism audit — every command composes documented components; goldens prove it | final screen-by-screen sweep |

Column: **"which components land when"** — components marked `[kbd]`/
`[static]` in `screen-spec.md` are keyboard/static contracts that must
not regress across these windows.

Implementation peers with the existing per-window themes:

1. Progress bar steps come from `engine.Progress` phases only (no
   guessed totals — S06 stops being a single transient line). **Done:
   `cli/internal/prompt/progress.go` (`ProgressGroup`) is the
   persistent step stack; percentage is `done/total` from real phase
   transitions.** Pending rows are reserved from the selected
   capability ids; rows match engine events by label, so they can never
   misattribute a phase.
2. Terminal title is TTY-gated and restored on exit (POSIX + Windows
   via kernel32; implemented in `cli/internal/prompt/title*.go`); off-TTY
   output stays deterministic. **Done.**
3. Determinism goldens extend to the new persistent S06 frame.
   **Done:** off-TTY S06 is one static line per phase (arrow line
   overtyped with the done/failed glyph), byte-stable and escape-free.

### v1.0.0 — Stable

At v1 freeze:

- The Stable surface list (ADR-0013) enumerates exactly the command/flag
  set in design spec §12.
- **`docs/cli/design-spec.md` and `screen-spec.md` are shipped docs** —
  a designer/contributor can reconstruct token-by-token intent.
- v1.x = new themes, multi-language SDK proofs, more ecosystems — all
  additive, all gated on the same design pillars.

---

## Sequencing rules (constraints)

1. **Additive before freezing.** Everything in S1–S2.1 is additive to
   existing Stable surface. Anything that *changes* a Stable command's
   behavior must land before the v1 freeze (v0.8.0) — after that it needs
   a minor bump + migration guide per ADR-0013's breaking-change bar.
2. **Tests/goldens:** existing `tests/integration` behavior is the
   contract; each design step must leave `go test ./cli/... ./core/...
   ./tests/...` (the 70s integration run) green and goldens unchanged
   unless the change is explicitly behavioral.
3. **No theme explosion before v1.** Two themes ($present §10.3) is the
   v1 number; the registry is extension.
4. **Docs stay home to the spec.** UI text changes land in
   `design-spec.md` first (spec §1.2, §7.1), then code — so the spec is
   the source of truth, not a periodic afterword.

---

## Definition of "design done" (v1 closure)

- [ ] All screens in `screen-spec.md` are implemented or marked
      deferred-with-ADR (none marked for the v1 window).
- [ ] Typed error taxonomy in place; every user-facing site uses it.
- [ ] `doctor` answers "which plugins / where from / is my binary
      staged?"
- [ ] Confirm step, capabilities guard, timing note shipped.
- [ ] A11y contract + contrast assertions in the theme registry; both
      themes render every S01–S15 screen to spec.
- [ ] Cross-platform matrix (§11) passes on all five v1 targets.
- [ ] **Design system shipped**: every screen in `screen-spec.md` maps
      to components in `design-system.md`; every command composes them;
      terminal title set/restored; persistent progress bar (S06) real
      and step-based.
