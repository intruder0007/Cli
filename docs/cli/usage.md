# CLI usage

## Interactive

```sh
lumo new
```

Running `lumo` with no arguments at all (including double-clicking
the binary on Windows) is the same as `lumo new`.

When run in a real terminal (stdin and stdout both TTYs), this is an
arrow-key wizard: `↑`/`↓` (or vim-style `j`/`k`) move the highlight,
`enter` confirms, `space` toggles a checkbox in the capabilities step,
`Ctrl+C`/`q` cancels. You can also **type to filter**: any letters
narrow the list by fuzzy subsequence match (e.g. `rbapi` finds
"REST API (node:http)"), `backspace` edits the filter, and `esc`
clears the filter first — cancelling only on a second `esc` (or
`Ctrl+C`/`q`). The available keys are shown in a hint line under every
menu. When stdin/stdout aren't both real terminals (piped input, CI,
scripts), it falls back automatically to plain numbered-list prompts —
see ADR-0007.

Up to seven prompts, in order (every menu is built from the plugins
actually installed — the wizard only offers what's discoverable, and
each step's options are filtered by the previous answers):

0. **Project name** — typed.
1. **Location** — typed, **always asked** (never silently defaulted —
   in particular, never the current working directory, which on
   Windows is the double-clicked `.exe`'s own folder). Pre-filled with
   the last-used location, editable every run, and **persisted**
   (`lumo config get projects-dir`) when you move past this step. First
   run defaults to `~/Projects` (or your home directory if that can't
   be determined).
2. **Theme** — `default` (color + icons) or `minimal` (plain text,
   `NO_COLOR`/screen-reader friendly). Whatever you pick here is
   **persisted** (`lumo config get theme`) and offered as the
   default next time.
3. **Project type** — the distinct project types of the installed
   template plugins (e.g. `Backend Service`), in discovery order.
4. **Language** — the languages available for the project type you
   picked (e.g. `Go`, `Node.js`; see
   [ADR-0009](../architecture/adr/0009-second-template-cross-language.md)).
5. **Framework** — the templates available for that project type +
   language pair, by their display names (e.g. `Go REST API Service`).
   The list is filtered by your choices, so a combination with no
   matching template can't be picked in the first place.
6. **Capabilities** (multi-select, checkboxes) — the installed
   capability plugins (e.g. `git-init`, `readme`); the step is skipped
   entirely when none are installed. `github-actions-ci` writes a Go
   workflow (`go build` + `go test`) and refuses non-Go projects with a
   clear error, so it only makes sense alongside the Go template.

If no template plugins are discoverable at all, the wizard fails fast
with a hint to run `lumo plugins list` / check `LUMO_PLUGIN_DIRS`
instead of asking questions nothing could resolve.

The CLI then resolves the matching template plugin, generates the project,
applies each selected capability in the order chosen, and prints a success
screen: a one-line summary (template used, file count, capabilities
applied), the files written, and next steps (e.g. `cd my-project && go
run .` or `cd my-project && npm start`). A failure prints an error screen
with a recovery hint where one is known.

Pass `--verbose` (or `-v`) to `lumo new` to print diagnostic
logging — plugin spawn, handshake result, timing, file counts — to
stderr as the run progresses. Useful when a run is slow or fails and the
error screen's hint isn't enough context.

## Non-interactive

For scripting, CI, or testing — this path is unaffected by the wizard's
theme persistence (never reads or writes the persisted theme):

```sh
lumo new my-project \
  --theme minimal \
  --project-type backend-service \
  --language go \
  --framework rest-api \
  --capabilities git-init,readme,github-actions-ci
```

### Target paths

The positional argument is a project name **or a target path**; the
project name (used by templates to name the module/service) is always
the path's final component:

- `lumo new my-app` — creates `./my-app` (relative to the CWD). This is
  the one non-interactive behavior the wizard's location step
  deliberately doesn't change: a script's working directory is part of
  its contract, so non-interactive runs keep resolving a bare name
  against the CWD.
- `lumo new ./work/app` — creates the app inside an explicit relative
  directory (parents are created as needed).
- `lumo new /abs/path/app` (Unix) or `lumo new C:\code\app` (Windows)
  — creates at an explicit absolute location.
- `lumo new ~/code/app` — `~` expands to the user's home directory on
  all platforms (`~/` and `~\` are both accepted).
- `lumo new my-app --dir ~/Projects` — an explicit, unambiguous
  alternative to a path-like positional: combines `--dir` and a bare
  project name into the target (`--dir` can't be combined with a
  path-like name — pick one form).

The project-name rules (start with a letter; letters, digits, `-`, `_`
only) apply to the final path component, so `lumo new ./x/2bad` is
rejected just like `lumo new 2bad`. Everything else in the path can be
arbitrary. This works identically for the answers-file form, where
`projectName` may hold a path.

or, from a file:

```sh
lumo new --answers answers.yaml
```

```yaml
# answers.yaml
projectName: my-project
theme: minimal
projectType: backend-service
language: go
framework: rest-api
capabilities: [git-init, readme, github-actions-ci]
```

The `projectName` value follows the same rules as the positional
argument: it may be a bare name or a target path (see
[Target paths](#target-paths)).

## Other commands

- `lumo plugins list` — lists discovered template and capability
  plugins (name, kind, version) from the local `templates/`/`plugins/`
  directories.
- `lumo plugins validate <plugin-dir>` — checks a plugin directory
  before shipping it: `plugin.json` must parse and pass
  `Manifest.Validate()`, and the entrypoint binary must spawn and pass
  the `plugin.initialize` identity/protocol cross-check against the
  on-disk manifest (the same fail-fast surface `new` uses, ADR-0008 —
  a stale or swapped binary fails here). Exits 0 when valid, 1 when
  not. The pre-release check for plugin/template authors.
- `lumo config get theme` / `lumo config set theme <name>` —
  read or explicitly set the persisted theme, without running the
  wizard.
- `lumo config get projects-dir` / `lumo config set projects-dir <path>`
  — read or explicitly set the persisted default project location
  (what the wizard's location step pre-fills), without running the
  wizard.
- `lumo doctor` — runs local health checks (plugin directory
  resolution, manifest validity, whether this binary has embedded
  fallback plugin assets and whether they're currently in use — see
  [ADR-0012](../architecture/adr/0012-universal-install-architecture.md))
  and prints a pass/fail summary with a recovery hint. Spawns no plugin
  subprocess — discovery only.
- `lumo version` — prints the CLI version, Go runtime version, and
  OS/arch (e.g. `lumo version v1.0.0 (go1.26.5, windows/amd64)`) —
  useful context to include in a bug report.
- `lumo help` / `lumo <command> -h` — top-level and
  per-command help.

## Exit codes

- `0` — success (for `plugins validate`: the plugin directory is valid).
- `1` — runtime failure: invalid answers (bad project name, unknown
  theme, missing required fields), an unresolvable template or
  capability, a plugin error (including a plugin that fails the
  handshake), a target directory that already exists and is non-empty,
  a missing/unreadable `--answers` file (or one that fails validation),
  and `plugins validate` on an invalid plugin directory.
- `2` — usage error: unknown command or flag, extra positional
  arguments to `new`, or combining `--answers` with a positional
  project name.
- `130` — cancelled with Ctrl+C (interactive wizard only).

## Accessibility notes

- `--no-color` and the `NO_COLOR` env var force the `minimal` theme's
  color behavior regardless of the selected theme.
- Every status the CLI reports (success, failure, "coming soon") has a
  text label — color/icons are additive, never the only signal.
- The arrow-key wizard is purely additive: anything it can do, the
  fallback line-based wizard and the non-interactive flags can also do —
  no functionality requires a fancy terminal. Type-ahead filtering is a
  navigation convenience only: every option stays selectable with the
  arrow keys and enter in both wizard modes.
