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
3. **Language** — `Go` or `Node.js` are selectable in V1 (see
   [ADR-0009](../architecture/adr/0009-second-template-cross-language.md) —
   the first two real proof points of "cross-language").
4. **Framework** — `REST API (net/http)` (Go) or `HTTP API (node:http)`
   (Node.js) are selectable in V1. The framework list isn't filtered by
   the language you picked — choosing a combination with no matching
   template (e.g. Go + `HTTP API`) fails cleanly with a hint to run
   `bootstrap plugins list`, same as any other unmatched combination.
5. **Capabilities** (multi-select, checkboxes) — `git-init`, `readme`,
   `github-actions-ci`. Go-project-specific today; only offered
   meaningfully alongside the Go template for now.

The CLI then resolves the matching template plugin, generates the project,
applies each selected capability in the order chosen, and prints a success
screen: a one-line summary (template used, file count, capabilities
applied), the files written, and next steps (e.g. `cd my-project && go
run .` or `cd my-project && npm start`). A failure prints an error screen
with a recovery hint where one is known.

Pass `--verbose` (or `-v`) to `bootstrap new` to print diagnostic
logging — plugin spawn, handshake result, timing, file counts — to
stderr as the run progresses. Useful when a run is slow or fails and the
error screen's hint isn't enough context.

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
- `bootstrap doctor` — runs local health checks (plugin directory
  resolution, manifest validity, whether this binary has embedded
  fallback plugin assets and whether they're currently in use — see
  [ADR-0012](../architecture/adr/0012-universal-install-architecture.md))
  and prints a pass/fail summary with a recovery hint. Spawns no plugin
  subprocess — discovery only.
- `bootstrap version` — prints the CLI version, Go runtime version, and
  OS/arch (e.g. `bootstrap version v0.2.0 (go1.22.5, windows/amd64)`) —
  useful context to include in a bug report.
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
