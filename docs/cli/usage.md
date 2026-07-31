# CLI usage

## Interactive

```sh
bootstrap new
```

When run in a real terminal (stdin and stdout both TTYs), this is an
arrow-key wizard: `↑`/`↓` (or vim-style `j`/`k`) move the highlight,
`enter` confirms, `space` toggles a checkbox in the capabilities step,
`Ctrl+C`/`Esc`/`q` cancels. When stdin/stdout aren't both real terminals
(piped input, CI, scripts), it falls back automatically to plain
numbered-list prompts — see ADR-0007.

Six prompts, in order:

0. **Project name** — typed.
1. **Theme** — `default` (color + icons) or `minimal` (plain text,
   `NO_COLOR`/screen-reader friendly). Whatever you pick here is
   **persisted** (`bootstrap config get theme`) and offered as the
   default next time.
2. **Project type** — only `Backend Service` is selectable in V1; other
   options are shown as "(coming soon)".
3. **Language** — only `Go` is selectable in V1.
4. **Framework** — only `REST API (net/http)` is selectable in V1.
5. **Capabilities** (multi-select, checkboxes) — `git-init`, `readme`,
   `github-actions-ci`.

The CLI then resolves the matching template plugin, generates the project,
applies each selected capability in the order chosen, and prints a success
screen with the files written and next steps (e.g. `cd my-project && go
run .`). A failure prints an error screen with a recovery hint where one
is known.

## Non-interactive

For scripting, CI, or testing — this path is unaffected by the wizard's
theme persistence (never reads or writes the persisted theme):

```sh
bootstrap new my-project \
  --theme minimal \
  --project-type backend-service \
  --language go \
  --framework rest-api \
  --capabilities git-init,readme,github-actions-ci
```

or, from a file:

```sh
bootstrap new --answers answers.yaml
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

## Other commands

- `bootstrap plugins list` — lists discovered template and capability
  plugins (name, kind, version) from the local `templates/`/`plugins/`
  directories.
- `bootstrap config get theme` / `bootstrap config set theme <name>` —
  read or explicitly set the persisted theme, without running the
  wizard.
- `bootstrap version` — prints the CLI version.
- `bootstrap help` / `bootstrap <command> -h` — top-level and
  per-command help.

## Accessibility notes

- `--no-color` and the `NO_COLOR` env var force the `minimal` theme's
  color behavior regardless of the selected theme.
- Every status the CLI reports (success, failure, "coming soon") has a
  text label — color/icons are additive, never the only signal.
- The arrow-key wizard is purely additive: anything it can do, the
  fallback line-based wizard and the non-interactive flags can also do —
  no functionality requires a fancy terminal.
