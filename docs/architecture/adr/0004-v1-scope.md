# ADR-0004: V1 scope is exactly one working template

## Status

Accepted

## Context

The platform is meant to eventually support many project types, languages,
and frameworks via plugins. Building several templates for V1 would spread
effort thin and delay proving the full pipeline (wizard → engine → plugin
host → generated, working output) end to end. The wizard steps themselves
(theme, project type, language, framework, capabilities) are still needed
in full even for one template, since they're the interaction contract the
plugin system is built around.

Candidates considered for the single V1 template: a TypeScript/React web
app, a Python CLI tool, a Go backend service, or a language-agnostic
"hello world." A Go + REST API backend service was chosen: it dogfoods the
same language as the core CLI, needs no bundler/frontend toolchain to also
get right, and is simple to hold to a "must actually build and pass tests"
bar.

## Decision

V1 ships exactly one template plugin: **Go + REST API backend service**
(`templates/go-rest-api`), plus exactly three capability plugins:
`git-init`, `readme`, `github-actions-ci`. The wizard's project type,
language, and framework steps present the full set of intended future
options, but only the V1 combination is selectable — others are shown as
"(coming soon)".

## Consequences

- "Done" for V1 has a concrete, testable bar: the generated project must
  `go build` and `go test` successfully, and each selected capability must
  produce its expected output.
- Every other project type/language/framework/capability combination is
  explicitly deferred (see [roadmap.md](../roadmap.md)), not silently
  unsupported — the wizard shows them so the plugin surface area is
  visible from day one.
- Because templates and capabilities are plugins (ADR-0002), adding the
  next template later requires no core changes — only a new plugin package
  and manifest.
