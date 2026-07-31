# ADR-0009: Second template (Node.js) — the first real cross-language proof

## Status

Accepted

## Context

V1 shipped a single template (`go-rest-api`) and a wizard whose language
step already listed Go, TypeScript, Python, and Rust — but only Go was
ever real; the platform is named "cross-language" while having only ever
generated Go projects. ADR-0004 deliberately deferred this ("adding a
template is a plugin, not a core change"), but the claim was never
actually exercised.

Adding a second template is also the natural forcing function for two
real manifest-schema gaps the original Phase 2 mandate named directly:
"dependencies" (addressed for capabilities in ADR-0008's `dependsOn`;
templates don't have an analogous need yet — no V1 template requires
another plugin) and "supported platforms" (genuinely needed once a
second template exists, since not every template will necessarily run
everywhere).

Language choice was constrained by what's actually installed and
verifiable in the environment building this: Node.js (v24) is present;
Python is not. This wasn't a preference call — it's the only language
choice that could be *proven* to work end-to-end rather than merely
claimed.

## Decision

- **`templates/node-rest-api`**: a second template plugin. Like
  `go-rest-api`, the *plugin* is written in Go (no Node.js SDK exists —
  see `roadmap.md`), but what it *generates* is a genuine Node.js
  project: `package.json`, `server.js` (`node:http`, zero dependencies),
  `server.test.js` (`node:test` + `node:assert` + global `fetch`, zero
  dependencies). Same quality bar as `go-rest-api` (ADR-0004): the
  generated project must actually run its own tests, standalone,
  offline — proven by `npm test` passing with **no `npm install` step
  at all**, since there's nothing to install.
- **`Manifest.SupportedPlatforms []string`** (templates only, optional;
  empty means all platforms): `core/registry.ResolveTemplate` filters
  out templates that don't list the current `runtime.GOOS`. Neither V1
  template needs this yet (Go and Node.js are both genuinely
  cross-platform), but the field and the filtering logic are real and
  tested, not just documented aspiration.
- **Wizard wiring**: `node` added to the language list, `http-api` added
  to the framework list, both directly selectable (not "coming soon").
  The framework list is *not* filtered by selected language in V1 —
  picking an invalid combination (e.g. Go + `http-api`) fails with the
  same `TemplateNotFoundError` + recovery hint any other unmatched
  combination already produces. A per-language framework list is a
  reasonable future improvement, not required to ship two real
  templates honestly.
- **CI**: `.github/workflows/ci.yml` gets an explicit
  `actions/setup-node@v4` step, rather than relying on whatever Node.js
  version `ubuntu-latest` happens to ship — a runner-image change should
  never be able to silently start skipping the cross-language proof.

## A real bug found while verifying this (not hypothetical)

The first version of `server.js` detected "am I the entrypoint" via
`import.meta.url === \`file://${process.argv[1]}\`` — a common Node.js
ESM idiom. It **silently fails on Windows**: `import.meta.url` is a
properly-encoded `file://` URL with forward slashes; `process.argv[1]`
is a raw OS path with backslashes there. The string comparison never
matches, so the server function was defined but never invoked — `npm
start` exited cleanly (code 0) without ever binding to a port, no error,
no log line. Caught only because the "must actually run" verification
step was followed through (start the server, curl it) rather than
stopping at "the test suite passes." Fixed with `pathToFileURL(
process.argv[1]).href`, which correctly normalizes for the current OS.

## Consequences

- The wizard steps documented since V1 ("only Go is selectable") are now
  genuinely two real options, one per major backend language family.
- `tests/integration` gains `TestEndToEndGenerateNodeRestAPI`, mirroring
  the Go test's rigor: real CLI invocation, real generated files, real
  `npm test` run, skipped gracefully (not failed) if `npm` isn't on
  PATH locally — but CI always has it via the new `setup-node` step, so
  it's never silently skipped there.
- `Makefile` and `ci.yml`'s module lists both include the new template.
- A third template (e.g. Python, once installable/verifiable in a build
  environment) is the natural next increment if this needs reinforcing
  further — not attempted now, to keep this phase's scope proportionate.
