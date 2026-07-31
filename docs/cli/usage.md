# CLI usage

## Interactive

```sh
bootstrap new
```

Walks through five prompts, in order:

1. **Theme** — `default` (color + icons) or `minimal` (plain text,
   `NO_COLOR`/screen-reader friendly).
2. **Project type** — only `Backend Service` is selectable in V1; other
   options are shown as "(coming soon)".
3. **Language** — only `Go` is selectable in V1.
4. **Framework** — only `REST API (net/http + chi)` is selectable in V1.
5. **Capabilities** (multi-select) — `git-init`, `readme`,
   `github-actions-ci`.

The CLI then resolves the matching template plugin, generates the project,
applies each selected capability in the order chosen, and prints a summary
with next steps (e.g. `cd my-project && go run .`).

## Non-interactive

For scripting, CI, or testing:

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
- `bootstrap version` — prints the CLI version.

## Accessibility notes

- `--no-color` and the `NO_COLOR` env var force the `minimal` theme's
  color behavior regardless of the selected theme.
- Every status the CLI reports (success, failure, "coming soon") has a
  text label — color/icons are additive, never the only signal.
