# Architecture overview

## Subsystem map

```
                       ┌─────────────┐
                       │     cli     │  interactive/non-interactive wizard
                       │ (stdlib)    │  (theme, project type, language,
                       └──────┬──────┘   framework, capabilities)
                              │ calls
                              ▼
                       ┌─────────────┐
                       │    core     │  engine, plugin host, config,
                       │             │  local registry
                       └──────┬──────┘
                              │ imports (wire types only)
                              ▼
                       ┌─────────────┐
                       │   sdk/go    │  plugin-author library:
                       │             │  wire types + sdk.Serve helper
                       └──────┬──────┘
                              │ implemented by
                 ┌────────────┼────────────┐
                 ▼            ▼            ▼
          templates/    plugins/builtin/  (future: third-party
          go-rest-api   git-init, readme,  plugins, any language,
                        github-actions-ci  same protocol)
```

`core` never imports `templates/` or `plugins/` packages — it discovers
their compiled binaries via `plugin.json` manifests and talks to them as
subprocesses over the protocol in [plugin-protocol.md](plugin-protocol.md).
The only code shared between `core` and plugin authors is `sdk/go`'s wire
types, which both sides depend on independently.

## Request flow (`bootstrap new`)

1. `cli` collects `Answers{Theme, ProjectType, Language, Framework,
   Capabilities[]}` — interactively (wizard) or from flags/`--answers` file.
2. `core/config` validates and normalizes the answers.
3. `core/registry` resolves the one template plugin matching
   `ProjectType`/`Language`/`Framework` (V1: always `go-rest-api`).
4. `core/plugin` spawns the template plugin, calls `plugin.generate`.
5. For each selected capability, in the order the user picked them,
   `core/plugin` spawns that capability plugin and calls `plugin.apply`.
6. `core/engine` aggregates `filesWritten`/`nextSteps` from every step and
   returns a summary; `cli` renders it (theme-aware).

## Module boundaries

| Module | May import | May NOT import |
|---|---|---|
| `cli` | `core` | `templates/*`, `plugins/*` |
| `core` | `sdk/go` (wire types only) | `templates/*`, `plugins/*`, `cli` |
| `sdk/go` | (stdlib only) | `core`, `cli`, any specific plugin |
| `templates/*`, `plugins/*` | `sdk/go` | `core`, `cli`, each other |

This is enforced by `go.mod` per module (Go modules cannot import a
package they haven't declared a dependency on), not just convention.

## Accessibility

The `default` theme uses color and icons; the `minimal` theme is plain
text, honors `NO_COLOR`/`--no-color`, and never encodes state (success/
failure/selection) in color alone — every state has a text label too. See
[ADR-0004](adr/0004-v1-scope.md) and `docs/cli/usage.md`.
