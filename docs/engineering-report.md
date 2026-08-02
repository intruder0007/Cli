# Engineering Report: the `cil` platform through v0.3.0

Final report for the six-phase engagement (Phases A–F), covering the
platform's architecture, what each phase shipped, the compatibility
surfaces that now exist, the audit that closed the engagement, and the
road ahead.

---

## 1. Executive Summary

`cil` (`bootstrap`) is a plugin-based project-scaffolding CLI with a
hardened JSON-RPC wire protocol, a published Go SDK, a documented
distribution contract, and an ADR-driven design process. Over six
phases the platform went from a working CLI (Phases A–B) to a
*verifiable* and *documented* one:

- **Phase A** pinned the public-surface compatibility policy
  (ADR-0013) that every later change is judged against.
- **Phase B** established `sdk/go` as the reference implementation of a
  language-neutral SDK spec (ADR-0014), with the wire protocol as its
  only coupling.
- **Phase C** added the pre-release extension check (`bootstrap plugins
  validate`), proving a plugin's manifest and binary agree before it is
  shipped.
- **Phase D** pinned the protocol with byte-exact golden transcripts on
  both sides of the wire, so any drift breaks CI.
- **Phase E** delivered the documentation suite (API reference,
  versioning guide, migration guide, tutorials) that makes the four
  Stable surfaces consumable.
- **Phase F** audited the whole codebase, fixed 25 findings (host
  lifecycle hardening, CLI surface guards, deterministic plugins,
  honest distribution docs), and left a written record
  (`docs/architecture/codebase-audit.md`).

All six phases shipped as merged PRs (#26–#31) with CI green — build,
vet, test, goldens, markdownlint, and the seven distribution-wrapper
verification jobs. The known gaps are explicitly tracked rather than
silent: no submodule tags yet (so `go install`/`go get` do not resolve),
and the embedded-plugin fallback exists only for pipeline-built
binaries.

## 2. Architecture Changes

The system is four cooperating layers, each with a clear boundary:

```text
        cli (command surface, wizard, embedded fallback)
              │
        core (config, engine, plugin host, registry, diag)
              │  JSON-RPC over stdio (plugin protocol)
        plugins/templates (V1 set: 2 templates, 3 capabilities)
              │
        sdk/go (reference SDK implementation of ADR-0014)
```

- **Phase A** added the compatibility policy layer (ADR-0013 +
  `api-compatibility.md`) — an inventory of every public surface with
  its stability stage, and the semver rules binding releases to those
  surfaces.
- **Phase B** added the SDK layer: `sdk/go/sdk` (Manifest, Plugin
  interfaces, requests/responses, `Serve`) and the language-neutral
  spec (`sdk-architecture.md`) with design notes for future SDKs.
- **Phase C** added `core/registry.LoadPluginDir` and
  `core/plugin.Host.Validate` — the validation path shared by `bootstrap
  plugins validate` and the fail-fast resolution in `new`.
- **Phase D** refactored `sdk.Serve` into an unexported
  `serveWithIO(reader, writer)` (public API unchanged) to enable
  byte-exact in-process wire testing, plus the `core/plugin/wiretest`
  scripted responder binary for the host side.
- **Phase E** added no runtime code — a documentation suite and index.
- **Phase F** hardened behavior without changing shapes: `Host.Stderr`
  wiring, response-id verification, `ok:false` handshake handling,
  ShutdownTimeout-bounded shutdown, capability dedupe, target-directory
  guard, Ctrl+C cancel paths, `--no-color` icons, deterministic
  `FilesWritten`, and the `github-actions-ci` language gate.

## 3. Public APIs

Four surfaces are declared **Stable** (ADR-0013, enumerated in
`docs/guides/api-reference.md`):

1. **Wire protocol** — `plugin.initialize` / `plugin.generate` /
   `plugin.apply` / `plugin.shutdown` over newline-delimited JSON-RPC
   on stdio; `protocolVersion` in the initialize handshake; error
   codes `-32601`, `-32602`, `-32700`, `-32000`. Byte-exact golden
   transcripts pin the lifecycle (Phase D).
2. **`sdk/go`** — `Manifest`, `Manifest.Validate`, `GenerateRequest`/
   `Response`, `ApplyRequest`/`Response`, `TemplatePlugin`,
   `CapabilityPlugin`, `Serve`. Declared Stable in its package doc.
3. **CLI commands** — `new` (interactive wizard, implicit
   non-interactive mode, `--answers`), `plugins list`, `plugins
   validate <dir>`, `config get/set theme`, `doctor`, `version`,
   `help`. Exit codes: 0 success, 1 runtime failure, 2 usage error,
   130 cancelled.
4. **Distribution wrapper contract** — 7 wrappers (npm, PyPI, Cargo,
   Homebrew, Scoop, Winget, Chocolatey) implementing the four-step
   protocol: resolve platform, fetch release archive, exec the
   embedded binary with stdio passthrough, forward argv and exit code.

**Experimental**: `core/*` packages and the theme registry
(`cli/internal/prompt` — internal, per ADR-0013).

## 4. SDK Architecture

`docs/architecture/sdk-architecture.md` (ADR-0014) is the
language-neutral spec: four abstractions — Manifest, Plugin,
Requests/Responses, Serve — with exact JSON-RPC bindings, package
layout, transport rules, and a compatibility strategy (SDKs are
additive; `sdk/go` is the reference implementation; no version
negotiation today, multi-version sketched for a future protocol bump).
Design notes exist under `sdk/{node,python,rust,future}/`; none beyond
`go` is implemented, per scope. The Go SDK is dependency-free (stdlib
only), so plugin authors can build against it offline; the wire tests
prove a hand-rolled protocol implementation can interoperate with it
(the Phase E tutorial demonstrates this).

## 5. Compatibility Guarantees

- **Wire**: changes require a `protocolVersion` bump and ADR-level
  process; mismatches fail loudly at the handshake (ADR-0008,
  ADR-0013). The goldens make drift a CI failure.
- **Go API**: `sdk/go` is Stable; `core/*` Experimental — breaking
  changes there are permitted but documented in the migration guide.
- **CLI**: commands, flags, exit codes, and output lines are part of
  the Stable surface.
- **Distribution**: the wrapper contract is Stable; wrappers forward
  argv and exit codes exactly.
- **Semver**: every release binds to the surfaces it affects (Phase E's
  `versioning.md` documents this for consumers).
- **Honesty**: the Phase F audit corrected overclaims — `go install`
  does not yet deliver the embedded fallback, no `--non-interactive`
  flag exists (implicit instead), and `go get` of the SDK awaits
  submodule tags. `docs/guides/migration-guide.md` records every
  user-visible breaking change so far.

## 6. ADRs Added/Updated

- **ADR-0013** (Phase A, Accepted) — public API compatibility policy.
- **ADR-0014** (Phase B, Accepted) — SDK architecture and design.
- Earlier records (0001–0012) established the core decisions the
  later phases build on: core language, plugin protocol (0004),
  release process (0006), CLI v2 experience (0007), protocol hardening
  (0008), second template (0009), distribution architecture (0010),
  summary contract (0011), universal install/embedded fallback (0012).

## 7. Files Modified

High-level footprint per phase (details in each PR):

- **Phase A** (#26): `docs/architecture/api-compatibility.md`,
  `adr/0013-*`, CHANGELOG.
- **Phase B** (#27): `sdk/go/**` (new module), `sdk/*/design-notes`,
  `adr/0014-*`, `docs/architecture/sdk-architecture.md`.
- **Phase C** (#28): `core/registry` (+`LoadPluginDir`),
  `core/plugin` (+`Host.Validate`), `cli/main.go` (+`plugins
  validate`), authoring docs, CHANGELOG.
- **Phase D** (#29): `sdk/go/sdk/sdk.go` (`serveWithIO` refactor),
  wire tests + 4 goldens, `.gitattributes`, `core/plugin/wiretest/`,
  protocol docs, CHANGELOG.
- **Phase E** (#30): `docs/README.md`, `docs/guides/*` (4 new guides),
  README, api-compatibility pointer updates, CHANGELOG.
- **Phase F** (#31): 25 files — `core/plugin/host.go` (+tests),
  `core/engine/engine.go` (+test), `core/config/config_test.go` (new),
  `cli/main.go`, `cli/internal/prompt/{wizard,theme,prompt}.go`,
  both templates, `plugins/builtin/{git-init,github-actions-ci}`,
  `tests/integration/integration_test.go` (+6 tests),
  `.github/workflows/ci.yml` (+`make build`), and 9 doc files.

## 8. Tests Added

- **Core**: `Answers.Validate` table; engine dependency ordering,
  fail-fast resolution, duplicate-selection collapse; registry
  valid/missing/invalid manifests; host handshake success/mismatch,
  `ok:false`, stale response-id, call timeouts, RPC errors, `finish()`
  kill-on-hang and cooperative clean exit, stderr wiring.
- **Wire (Phase D)**: 4 golden transcripts — host→SDK (generate,
  apply, full lifecycle with real responder subprocess through
  recording pipes) and SDK→host (serve lifecycle, error contract),
  byte-exact for `-32601`/`-32000`, structural for
  `-32700`/`-32602`.
- **Integration (black-box)**: end-to-end Go (all 3 capabilities,
  generated project builds and passes its own tests), end-to-end
  Node (cross-language proof, `npm test`), embedded-fallback
  end-to-end, `plugins validate` (valid / identity mismatch / invalid
  manifest), CLI surface (extra positionals, answers+positional,
  non-empty target dir, unknown themes, config-set), and the
  github-actions-ci language gate.
- **CLI unit**: plugin-dir discovery, embedded override isolation,
  version-scoped cache dir.

## 9. Documentation Added

- `docs/README.md` index (Phase E); `docs/guides/{api-reference,
  versioning, migration-guide, tutorials}`; `docs/architecture/
  sdk-architecture.md`; "Wire protocol stability" + regeneration
  commands in `plugin-protocol.md`; authoring guides extended for
  validation (Phase C); `docs/architecture/codebase-audit.md` (Phase
  F); exit-code contract in `usage.md`; honest install-status docs
  across README, `distribution/go/README.md`,
  `distribution-protocol.md`, `roadmap.md`, and both tutorials.

## 10. Technical Debt

- **No submodule tags** — `cli/vX.Y.Z` and `sdk/go/vX.Y.Z` do not
  exist; `go install`/`go get @latest` fail to resolve.
- **`go install`-built binaries lack the embedded fallback** — assets
  are staged at build time into a gitignored dir; only the release
  pipeline embeds them.
- **Old-version embedded-cache GC** — version-scoped dirs accumulate.
- **Capability conflict detection** — ordering exists (ADR-0008),
  conflict detection does not.
- **`make build` on Linux** leaves extensionless binaries in plugin
  source dirs (`.gitignore` covers `*.exe` only).
- **No live distribution publishing** — 7 wrappers are built and
  CI-verified but unpublished.

## 11. Risks

- **Docs drift** — mitigated by CI lint, goldens, and the audit
  process; the Phase F doc corrections show how fast claims decay.
- **Protocol evolution** — single-version negotiation means a protocol
  bump breaks older plugins; multi-version serving is deferred until
  needed (documented, ADR-0013 gate).
- **Single reference SDK** — `sdk/go` is the only implementation; the
  spec's correctness is validated only by the golden transcripts and
  the hand-written tutorial client.
- **Windows-specific behaviors** — e.g. `git commit` identity in CI,
  raw-mode terminal handling; both are covered by tests/comments but
  remain platform-sensitive spots.

## 12. Recommendations for v0.4.x

1. **Tag the submodules** (`cli/v0.4.0`, `sdk/go/v0.4.0`) and re-run
   the documented `go install`/`go get` commands to unblock the
   distribution story.
2. **Decide the `go install` embedded-fallback gap** — either commit
   prebuilt assets (against the current gitignore design) or document
   `go install` as "binary only" permanently (the Phase F docs already
   say this).
3. **Add one more capability/test pair per release** to keep the
   ecosystem-loop proof honest; the `github-actions-ci` language gate
   is the template for capability-answer validation.
4. **Ship a real release** and publish at least one wrapper (npm is the
   lowest-friction credential) to retire the "built but unpublished"
   debt.
5. **Extend the goldens** whenever the wire changes, and consider
   fuzzing the host's JSON-RPC reader.

## 13. Long-Term Roadmap to v1.0

Tracked in `docs/architecture/roadmap.md`. v1.0 criteria: stable wire
protocol with multi-version negotiation if a bump lands, published
distribution (>=1 registry), submodule tags + working `go install`,
capability conflict detection, and at least two language SDKs or an
explicit decision to stay Go-only. Deferred items already scoped:
workspace/monorepo generation, capability-to-capability visibility,
theme plugins (public theme API), localization, auto-update, and
telemetry (which must never be silent).

## 14. Conclusion

The platform is now *specified, implemented, verified, and
documented* to a standard where every claim is either tested (goldens,
integration, CI) or explicitly flagged as residual. The six-phase
structure — policy, SDK, validation, wire-proof, docs, audit — left
each phase independently mergeable and CI-green, and the Phase F audit
closed the loop by auditing the earlier phases' own claims. The
remaining gaps are named, tracked, and unblocked by mechanical steps
(tagging, publishing) rather than by architectural risk.
