# Lumo — AI context report

> Generated 2026-08-02. A self-contained snapshot of the whole project,
> written so a fresh LLM (ChatGPT or other) can absorb full context
> before helping. Read sections 2–6 for the model; 7–14 as reference.
> Current commit on `main`: `872c962` (pushed 2026-08-02).

---

## 1. What Lumo is

**Lumo** is a **plugin-based project scaffolding platform**: an orchestrator CLI
that generates a working, production-shaped new project from one interactive
command. It is **not** a framework replacement — it composes plugins.

- **Repo:** `github.com/intruder0007/Lumo` (renamed from `github.com/intruder0007/Cli`;
  the product was renamed "Cli"/`bootstrap` → "Lumo" in 2026-08).
- **Language:** Go (stdlib-first; the only non-stdlib dep is `golang.org/x/term`,
  isolated to `cli`).
- **Status:** V1 feature-complete, **pre-1.0.0 release**. `main` has the full V1
  today; the v1.0.0 release is the end-of-mission decision, not yet cut.
- **Releases so far:** v0.1.0 (2026-07-31), v0.1.1 (2026-07-31, plugin-dir dedup),
  v0.2.0, v0.3.0 (the npm interim package was published against v0.3.0 on 2026-08-02).

### Principles

Offline first (no network to generate a project). Plugin first (templates and
capabilities are plugins; the core never hard-codes what it can generate).
Clean boundaries (each subsystem is an independently buildable Go module; they
talk over small explicit interfaces). Cross-platform (Linux/macOS/Windows).
Accessible (`NO_COLOR`, a screen-reader-friendly `minimal` theme, state never
encoded in color alone). Production quality (what V1 generates builds and
passes its own tests — not a stub).

### Mission phases (in order; current is npm migration → then the rest)

1. Rebrand `Cli`/`bootstrap` → `Lumo` (done).
2. UX redesign (done).
3. Registry-driven wizard architecture (done).
4. **npm identity migration — active.** Audit + repair done; the actual
   `lumo-cli@1.0.0` publish is **blocked on the v1.0.0 GitHub release**.
5. Security review (next).
6. First-run wizard / onboarding polish (planned).
7. Engineering reports (planned; some exclusion docs still carry the old name).
8. **v1.0.0 release decision** (end of mission — cuts the release that
   unblocks the npm publish and the wrapper re-verification).

---

## 2. Repository layout & module boundaries

Single repository with **long-lived per-subsystem branches merged into `main`
via PR** (convention documented in `README.md`/`CONTRIBUTING.md`).
One recent exception: commit `872c962` (the rebrand + UX + architecture + npm
work) was pushed **directly to `main`** on 2026-08-02 — a deviation from the
PR-first convention, recorded here honestly.

| Branch | Contains |
|---|---|
| `main` | Integration branch; releases tagged here |
| `architecture` | ADRs, architecture docs, repo scaffolding |
| `cli` | The interactive/non-interactive CLI |
| `core` | Orchestration engine, plugin host, config, local registry |
| `sdk` | `sdk/go` — plugin-author library |
| `templates` | Template plugins (`go-rest-api`, `node-rest-api`) |
| `plugins` | Built-in capability plugins (`git-init`, `readme`, `github-actions-ci`) |
| `tests` | Integration/golden tests + CI workflow |

### Module boundaries (enforced by `go.mod`, not just convention)

| Module | May import | May NOT import |
|---|---|---|
| `cli` | `core` | `templates/*`, `plugins/*` |
| `core` | `sdk/go` (wire types only) | `templates/*`, `plugins/*`, `cli` |
| `sdk/go` | (stdlib only) | `core`, `cli`, any specific plugin |
| `templates/*`, `plugins/*` | `sdk/go` | `core`, `cli`, each other |

`go.work` ties the modules together for local development; `tests` is its own
module driving black-box integration.

---

## 3. Architecture

```text
            ┌─────────────┐
            │     cli     │  interactive/non-interactive wizard
            │ (stdlib)    │  (theme, project type, language,
            └──────┬──────┘   framework, capabilities)
                   │ calls
                   ▼
            ┌─────────────┐
            │    core     │  engine, plugin host, config, local registry
            └──────┬──────┘
                   │ imports (wire types only)
                   ▼
            ┌─────────────┐
            │   sdk/go    │  wire types + sdk.Serve helper
            └──────┬──────┘
                   │ implemented by
          ┌────────┼────────┐
          ▼        ▼        ▼
   templates/  plugins/builtin/  (future: third-party plugins, any language)
```

`core` **never imports** `templates/` or `plugins/` packages — it discovers
their compiled binaries via `plugin.json` manifests and talks to them as
subprocesses. The only code shared between `core` and plugin authors is
`sdk/go`'s wire types, depended on independently by both sides.

### Request flow (`lumo new`)

1. `cli` collects `Answers{Theme, ProjectType, Language, Framework,
   Capabilities[]}` — interactively (wizard) or from flags/`--answers` file.
2. `core/config` validates and normalizes the answers.
3. `core/registry` resolves the one template plugin matching
   `ProjectType`/`Language`/`Framework`.
4. `core/plugin` spawns the template plugin, calls `plugin.generate`.
5. For each selected capability, in the order the user picked them (with
   `dependsOn` ordering, ADR-0008), `core/plugin` spawns it and calls
   `plugin.apply`.
6. `core/engine` aggregates `filesWritten`/`nextSteps` from every step and
   returns a summary; `cli` renders it (theme-aware). An additive
   `Engine.Progress(phase, done)` callback drives the spinner/phase UI.

### Wire protocol (Stable surface, ADR-0013)

- Subprocess + **line-delimited JSON-RPC 2.0** between host and plugin.
- Methods: `plugin.initialize` (identity/protocol handshake — mismatch rejected),
  `plugin.generate` (templates), `plugin.apply` (capabilities), `plugin.shutdown`.
- One `protocolVersion` per release; multi-version negotiation deferred.
- Pinned by **byte-exact golden transcripts** in
  `core/plugin/testdata/*.golden`, `sdk/go/sdk/testdata/*.golden`
  (regenerable via `-update`).

### Registry & discovery

- **Local directory scan only** (remote registry/marketplace is a deliberate
  V1 non-goal — `core/registry`'s interface is written so a remote source can
  be added later without changing callers).
- Lookup via `plugin.json` manifests next to each plugin binary.
- `LUMO_PLUGIN_DIRS` env var, `lumo doctor` for diagnosing setup problems.
- Default discovery: `templates/` and `plugins/builtin/` next to the binary.

### Embedded fallback (ADR-0012)

- The release pipeline **stages the V1 plugin set into `cli`'s own embed
  directory** (`cli/internal/embedded/assets`) via `go:embed` before building
  each platform's `lumo` binary, so the binary is self-sufficient (works
  without sibling `templates/`/`plugins/` directories — e.g. `go install`).
- Cache scoped per version (`os.UserCacheDir()/lumo/<version>/`); upgrading
  never serves stale plugins. No GC of old-version caches yet.
- The staged assets are gitignored; on the dev box they're staged manually.

---

## 4. CLI surface

### Commands

- `lumo new` — the wizard (interactive, or `--answers file.yaml` /
  flags for non-interactive).
- `lumo` with no args — also starts the wizard (so double-clicking the
  binary on Windows opens the wizard). Under piped stdin it hits the EOF
  path and fails with "project name is required" (exit 1). `lumo help`
  still prints the command reference.
- `lumo doctor` — health check for plugin-discovery setup problems.
- `lumo plugins list` — list discovered template/capability plugins.
- `lumo plugins validate <dir>` — pre-release extension check:
  validates a plugin dir's manifest and proves the entrypoint binary spawns
  and passes the `plugin.initialize` identity/protocol cross-check, without
  generating anything. Exit 0 = valid.
- `lumo version`, `lumo help`.

### Wizard UX (registry-driven since 1.0.0, ADR-0007)

- 5 steps: **theme → project type → language → framework → capabilities**
  (capabilities step skipped when none installed; capability step is the 5th,
  so labels show "Step N of M" dynamically).
- **Every menu is built from installed plugins** via `prompt.WizardSpec`
  (`core/registry.Discover()` → `wizardSpecFromDiscovery` in `cli/main.go`).
  Each step filters the next (language by project type, framework by type +
  language). Hardcoded option lists were deleted; a combination with no
  matching template can't be picked in the first place.
- **No template plugins discoverable → fail fast** with a hint
  ("no template plugins found — run `lumo plugins list`…").
- **Type-ahead fuzzy search** in every menu (subsequence match; `backspace`
  edits; `esc` clears filter first, cancels on second press; a hint line
  shows the keys). `pushbackReader` fix means a bare ESC no longer swallows
  the next keystroke.
- **Spinner + phase progress**: a plan line, then one live spinner per phase
  on a terminal (braille frames in `default`, ASCII in `minimal`), degrading
  to plain `- phase` lines under pipes/CI (no escape codes off a TTY).
- **Design tokens** on `Theme`: `Primary`/`Accent`/`Warn`/`Border` helpers
  with per-theme spinner frames; the whole CLI renders through semantic colors,
  keeping the `NO_COLOR`/`minimal` contract intact.
- **Screens**: banner "Lumo — a new project, ready in seconds"; tree
  success screen (├─/└─ glyphs, `+` in minimal) with arrow-marked next steps;
  error screens get breathing-room spacing.

### Themes

- `default` (color + icons) and `minimal` (plain text, honors `NO_COLOR`/
  `--no-color`, never encodes state in color alone).
- Theme registry (`RegisterTheme`/`GetTheme` in `cli/internal/prompt`) is a
  real in-process seam but lives in `internal/`, so it's **Experimental**,
  not public (ADR-0013).

### Exit codes

`0` success · `1` runtime/validation errors · `2` usage errors
(e.g. non-empty target dir, extra positionals, `--answers` + name) ·
`130` Ctrl+C (matches between TUI and line-based wizard).

### Flags of note

`--answers <file>` (non-interactive), `--no-color`, `--verbose`,
`--language`, `--framework`, etc. `--non-interactive` does **not** exist
(docs that claimed it were corrected).

---

## 5. Plugins & templates inventory

**Templates** (each a plugin implementing `plugin.generate`):

- `templates/go-rest-api` — Go REST API service (`net/http`).
- `templates/node-rest-api` — Node.js HTTP API service (`node:http`,
  zero npm dependencies, cross-language proof per ADR-0009).

**Capability plugins** (each implements `plugin.apply`, ordered via
`dependsOn`, ADR-0008):

- `plugins/builtin/git-init` — `git init` + initial commit.
- `plugins/builtin/readme` — generates README.
- `plugins/builtin/github-actions-ci` — GitHub Actions workflow (Go projects).

Authoring guides: `docs/plugins/authoring.md`, `docs/templates/authoring.md`,
tutorials in `docs/guides/tutorials.md`. A new stack = "drop in a template
plugin" — no core change.

---

## 6. SDK

- **`sdk/go`** — the only SDK shipped; wire types + `sdk.Serve(yourPlugin)`
  JSON-RPC transport. Stable surface (ADR-0013). Package doc declares its
  Stable status.
- **Language-neutral SDK spec** (`docs/architecture/sdk-architecture.md`,
  ADR-0014): four abstractions (Manifest, Plugin, Requests/Responses, Serve),
  exact protocol bindings, transport rules, compatibility strategy. Design
  notes under `sdk/{node,python,rust,future}/` — **only `sdk/go` is
  implemented**. Each future SDK is an additive phase with CI-verified
  clean-machine round-trip (same bar as wrappers).

---

## 7. Distribution

Eight ecosystems; wrappers are **thin launchers** that resolve platform →
fetch the release archive as a whole → exec `lumo` with stdio/argv/exit-code
passed through unmodified (never reimplement the CLI). Contract:
`docs/architecture/distribution-protocol.md`.

| Ecosystem | Status |
|---|---|
| npm | `bootstrap-cli-dev@0.3.0` **live + deprecated** (interim, old name, 2026-08-02); `lumo-cli@1.0.0` publish pending the v1.0.0 release |
| Go | "solved" via ADR-0012 embedded fallback (release archives / install scripts supported; `go install` not yet recommended) |
| PyPI | built, CI-verified, **not published** |
| Cargo | built, CI-verified, not published (lowest priority) |
| Homebrew | built, CI-verified, not published |
| Scoop | built, CI-verified, not published |
| Winget | built, verified, not published |
| Chocolatey | built, CI-verified, not published |

### Release archive layout

`lumo[.exe]` + `templates/` + `plugins/builtin/` per platform, named
**`lumo_${VERSION}_${GOOS}_${GOARCH}.{tar.gz,zip}`** + **`SHA256SUMS.txt`**,
produced by `.github/workflows/release.yml` (cross-compiled matrix:
linux/darwin × amd64/arm64 + windows/amd64).

### npm in detail

- **Interim package `bootstrap-cli-dev@0.3.0`** published 2026-08-02 under the
  old name (ADR-0015). **Deprecated 2026-08-02** with the verified message
  (registered on the registry, confirmed via cache-busted query):
  *"Renamed to lumo-cli: the successor package publishes as lumo-cli@1.0.0
  with the platform v1.0.0 release. Until that release this interim package
  continues to work; afterwards install lumo-cli instead."*
- **`lumo-cli@1.0.0`** is the renamed, version-synced successor (ADR-0016).
  The wrapper version **always equals** the `lumo` release it launches
  (ADR-0015), enforced by three guards: a version-sync check in `release.yml`'s
  `publish-npm` job, the `prepublishOnly` → `scripts/verify-release.js` guard
  (fails `npm publish` unless every `lumo_v1.0.0_*` archive + `SHA256SUMS.txt`
  exist), and postinstall's version-keyed idempotency marker.
- **postinstall** (`scripts/postinstall.js`) downloads the platform archive,
  verifies SHA-256 against `SHA256SUMS.txt`, extracts just the `lumo[.exe]`
  binary into the installed package's `.bin/`. Idempotency marker skips
  re-download; a deleted/corrupted binary always re-downloads. If install
  scripts are skipped (`--ignore-scripts`), the shim prints an actionable
  missing-binary error and exits 1.
- **Publish is automatic** via `release.yml`'s `publish-npm` job (runs only
  after the GitHub release exists), gated on the `NPM_TOKEN` repo secret
  (an npm *automation* token). Provenance signing via `npm publish --provenance`
  with GitHub OIDC. npm account `intruder214` (2FA).
- **`lumo` is taken** on npm (an unrelated WebGL library) — that's why the
  wrapper is `lumo-cli`. `lumo-cli` was unclaimed at audit time.

### ⚠️ Known state: wrapper URLs vs the not-yet-cut v1.0.0 release

The 6 non-npm wrappers (homebrew/scoop/winget/chocolatey/cargo/pypi) were
**forward-bumped to expect v1.0.0 assets** (`lumo_v1.0.0_*` at the `v1.0.0`
tag). Those assets don't exist yet — the latest release is **v0.3.0**, which
contains the **old-named** `cli_v0.3.0_*` archives. Verified 2026-08-02:
`lumo_v1.0.0_windows_amd64.zip` → **404**; `cli_v0.3.0_windows_amd64.zip`
and `SHA256SUMS.txt` → **200**. So `distribution-verify.yml` (which installs
each wrapper on a clean runner) **would fail on all wrapper jobs today** —
it has no `push` trigger (deliberately, to avoid racing the release pipeline)
and runs only on PRs touching `distribution/**` + manual dispatch. This is
the expected pre-release state, resolved when the **v1.0.0 release is cut**.

---

## 8. Governance & tooling

- **ADR process**: 16 ADRs (`0001`–`0016`), all statused **Accepted**, in
  `docs/architecture/adr/`. Index below.
- **Branch workflow**: long-lived per-subsystem branches → PR → `main`
  (documented; the recent `872c962` direct push is a recorded exception).
- **CHANGELOG**: [Keep a Changelog](https://keepachangelog.com/) format;
  `[Unreleased]` section carries Added/Changed/Deprecated/Fixed.
- **markdownlint**: `.markdownlint.json` at root; **MD013** (line length) and
  **MD060** (no-multiple-h1s) disabled. Run via
  `npx markdownlint-cli@0.43.0 <files> --config .markdownlint.json`.
- **Rebrand exclusions** (intentionally kept with old identity):
  `docs/{project-report,engineering-report,engineering-report-npm}.md`,
  `docs/architecture/codebase-audit.md`, and everything under
  `docs/architecture/adr/`. New ADRs/docs use the Lumo identity.

### CI workflows (`.github/workflows/`)

- `ci.yml` — `make build`, unit tests.
- integration workflow (in `tests/`) — black-box end-to-end.
- `distribution-verify.yml` — clean-machine install per ecosystem
  (PR on `distribution/**` + manual; **no push trigger** by design).
- `release.yml` — on `vX.Y.Z` tag: build archives + `SHA256SUMS.txt` →
  GitHub release → `publish-npm` job (gated on `NPM_TOKEN`) with version-sync
  check, `prepublishOnly` guard, provenance, clean-prefix smoke.
- `npm-verify-published.yml` — manual (`workflow_dispatch`) clean-machine
  check of the **published** npm package on Linux + Windows. **Must not be
  run before `lumo-cli@1.0.0` exists** — it installs `lumo-cli@latest`.

### Release process

Tag `vX.Y.Z` → `release.yml` → archives `lumo_vX.Y.Z_*` + `SHA256SUMS.txt` →
GitHub release → `publish-npm` (if `NPM_TOKEN` present) publishes `lumo-cli`
with provenance. Runbook, rollback, deprecation: `docs/guides/releasing.md`.
Version bumps and the changelog entry land in the same commit.

---

## 9. Testing

- **Unit tests** per module (`go test ./...`): `spec_test.go`
  (WizardSpec/filtering/humanize), `spinner_test.go`, `menu_test.go` (fuzzy
  search + ESC behavior), `engine_test.go` (Progress phases + failRunner),
  host lifecycle tests, registry tests, config tests.
- **Wire-protocol golden transcripts** (Phase D): byte-exact fixtures pin the
  full JSON-RPC lifecycle on both sides; regenerable with `-update`.
- **Integration** (`tests/integration`, black-box): bare-wizard starts the
  wizard; `--answers` generation; exit codes 1/2/130; `plugins validate`;
  wizard piped fallback generates a real project; no-templates fail-fast.
  **Last full run: PASS, ~63 s.**
- **Distribution verify**: per-ecosystem clean-machine install via the real
  wrapper path (currently blocked on the v1.0.0 release — see §7).

### Verification commands (dev box, Windows + pwsh)

```powershell
# Unit + integration
go test ./...
go test -count=1 ./tests/...      # ~63s integration

# Build: stage plugin binaries into cli/internal/embedded/assets, THEN build.
# A plain `go build ./cli` (without stage-embedded) embeds no plugin assets
# and fails to generate projects outside a source checkout — use make build
# (make build), or on Windows run the stage-embedded steps by hand, or just
# `go run ./cli` from the repo root where templates/ is a sibling directory.
go build -ldflags "-X main.version=v0.4.0" -o bin/lumo.exe ./cli

# Lint
npx markdownlint-cli@0.43.0 <files> --config .markdownlint.json

# npm pack smoke (no release needed for the pack itself)
cd distribution/npm; npm pack --dry-run
node scripts/verify-release.js    # must fail pre-v1.0.0 (404s)
```

---

## 10. ADR index (`docs/architecture/adr/`, all Accepted)

| # | Title |
|---|---|
| 0001 | Core language (Go) |
| 0002 | Plugin protocol (subprocess + JSON-RPC, local discovery) |
| 0003 | Branching workflow (per-subsystem branches → PR) |
| 0004 | V1 scope |
| 0005 | License (MIT) |
| 0006 | Release process |
| 0007 | CLI v2 experience (themes, wizard, design-token seam) |
| 0008 | Plugin architecture hardening (`dependsOn`, handshake, timeouts, stderr) |
| 0009 | Second template, cross-language (Node.js REST API) |
| 0010 | Distribution architecture (wrapper contract) |
| 0011 | Developer experience polish |
| 0012 | Universal install architecture (go:embed fallback) |
| 0013 | API compatibility policy (Stable/Experimental surfaces) |
| 0014 | SDK foundation (language-neutral spec) |
| 0015 | First published wrapper — npm package identity & release coupling |
| 0016 | npm identity migration — `bootstrap-cli-dev` → `lumo-cli` |

---

## 11. History / work log

- **2026-07-31** — v0.1.0 (initial release); v0.1.1 (`plugins list` dedup fix).
- **~2026-08-01** — v0.2.0; **Phase B** distribution wrappers (7/8 ecosystems,
  PR #23); **Phase A** universal install / ADR-0012 (PR #22).
- **2026-08-01/02** — v0.3.0 cut; all 7 wrappers repointed at v0.3.0 release
  assets; npm interim package `bootstrap-cli-dev@0.3.0` published
  (ADR-0015, first live channel).
- **Engineering phases A–F** (per `docs/engineering-report*.md`):
  - **A** — public API compatibility policy (ADR-0013).
  - **B** — SDK foundation (ADR-0014, `sdk-architecture.md`).
  - **C** — `lumo plugins validate` (manifest + binary identity cross-check).
  - **D** — byte-exact wire-protocol golden transcripts (host + SDK).
  - **E** — documentation suite (index, API reference, versioning, migration,
    tutorials).
  - **F** — codebase audit fixes (host lifecycle, CLI surface guards,
    deterministic plugins, doc corrections) — `docs/architecture/codebase-audit.md`.
- **Engineering report** (14 sections) — PR #32.
- **Bare-wizard fix** — bare `lumo` starts the wizard instead of printing
  help (PR #33, `089ef8c`).
- **2026-08-02 (uncommitted work then pushed as `872c962`):**
  - **Rebrand** `Cli`/`bootstrap` → **Lumo** everywhere (CLI sources, docs,
    distribution wrappers; intentional exclusions listed in §8).
  - **UX redesign**: design tokens, fuzzy menu search, spinner/progress
    phases, banner/success/error screen polish, wizard step labels.
  - **Registry-driven wizard** (ADR-0007 realized): `prompt.WizardSpec` from
    plugin discovery, per-step filtering, no-templates fail-fast; new tests
    (`spec_test.go`, `spinner_test.go`, fuzzy `menu_test.go`, engine
    progress tests) + new integration tests (piped-wizard generates a project,
    no-templates fails fast); full suite green; live piped smoke verified.
  - **npm identity migration phase**: corruption audit (single-letter
    substitutions: `aumo` repo URLs in `package.json`; "hhe"/"hhin"/`NPM_hOKEN`/
    `EBADPLAhFORM` in the npm README; `license = "MIh"` in cargo/pypi) and
    stale "lumo-cli@0.3.0 is published" claims — all repaired; final
    corruption sweep: zero matches; `npm pack` smoke (5 files, 6.0 kB);
    `verify-release.js` guard verified (6× 404 pre-v1.0.0); install-path
    smokes (postinstall fails loudly, `--ignore-scripts` shim error
    actionable); **`bootstrap-cli-dev@0.3.0` deprecated** on the registry;
    new docs `docs/architecture/npm-identity-migration.md` (audit record) +
    **ADR-0016**.
- **`872c962` pushed directly to `main`** 2026-08-02 (101 files, +2579/−717).

---

## 12. Current state (verified 2026-08-02, commit `872c962`)

- `main` is clean (nothing uncommitted after `872c962`).
- **Last full integration run**: PASS, ~63 s.
- **Corruption sweep** over the whole repo: zero matches.
  `package.json` / `scoop/lumo.json` parse; `Cargo.toml` validates
  (`cargo metadata --no-deps` exit 0); markdownlint clean on touched docs.
- **npm pack**: exactly 5 files (README, `bin/lumo.js`, `package.json`,
  `scripts/postinstall.js`, `scripts/verify-release.js`), 6.0 kB.
- **Registry**: `bootstrap-cli-dev@0.3.0` deprecated; message verified.

---

## 13. Next steps & blockers

1. **Security review** — next mission phase.
2. **First-run wizard / onboarding polish** — planned.
3. **v1.0.0 release decision** (end of mission). Cutting the `v1.0.0` tag
   triggers `release.yml` → produces `lumo_v1.0.0_*` archives + `SHA256SUMS.txt`
   → `publish-npm` publishes `lumo-cli@1.0.0` with provenance. **This unblocks:**
   - the 6 wrapper 404s (§7) resolve;
   - `distribution-verify.yml` goes green again;
   - run `npm-verify-published.yml` (only *after* the publish);
   - flip status docs to "published": `distribution/npm/README.md`,
     `distribution/README.md`, `docs/architecture/roadmap.md`,
     `docs/guides/releasing.md`.
4. **Engineering reports** — `docs/{project-report,engineering-report,
   engineering-report-npm}.md` still carry the old name (rebrand exclusions);
   refresh per the earlier plan.
5. **gofmt**: pre-existing unformatted files (`config.go`, `config_test.go`,
   `theme_test.go`) are left untouched by design — be aware before
   `gofmt -l` reports them.
6. **6 remaining ecosystems publishing** (PyPI/Cargo/Homebrew/Scoop/Winget/
   Chocolatey) — each an explicitly-confirmed future step (credential or PR).

---

## 14. Agent environment & gotchas (this dev box)

- **OS / shell:** Windows 11, PowerShell 7 (`pwsh`). No `make` locally (the
  `Makefile` is for CI; local build is `go run ./cli` from the repo root, or
  `go build ... -o bin/lumo.exe ./cli` after staging embedded assets — see §9).
- **Toolchain:** Go 1.26.5, Node v24.13.0, npm 11.6.2. No `rg`, `python`, or
  globally-installed `markdownlint` — use `npx markdownlint-cli@0.43.0`.
- **Embedded staging** is gitignored and manual on a dev box; for real
  binaries use the release pipeline. Embed dir: `cli/internal/embedded/assets`.
  A binary built without staging embeds no plugin assets and cannot generate
  projects outside a source checkout — this is the "lumo.exe does nothing but
  bootstrap.exe works" failure mode (staged binaries embed the V1 plugin set).
- **Git line endings**: autocrlf warns "LF will be replaced by CRLF" — harmless;
  golden transcripts are pinned to LF via `.gitattributes`.
- **npm auth**: automation token in `~/.npmrc` and a GitHub `NPM_TOKEN`
  secret. **Never print these values.** Account `intruder214`, 2FA on.
- **Conventions**: don't commit unless explicitly asked; commit messages
  follow the repo's conventional style (`feat:`, `fix:`, `docs:`, `chore:`).
  Direct pushes to `main` are unconventional (one recorded exception above).
- **Naming**: use **Lumo** / `lumo-cli` / `github.com/intruder0007/Lumo`
  in new content; keep the rebrand-excluded docs (§8) as-is.

---

### Quick links (read these next for depth)

- `README.md`, `docs/README.md` — orientation & doc index.
- `docs/architecture/overview.md` — subsystem map + request flow.
- `docs/architecture/plugin-protocol.md` — the wire protocol (Stable).
- `docs/architecture/distribution-protocol.md` — the wrapper contract.
- `docs/architecture/api-compatibility.md` — Stable/Experimental surfaces.
- `docs/architecture/roadmap.md` — what's deferred beyond V1.
- `docs/guides/releasing.md` — release runbook.
- `docs/architecture/npm-identity-migration.md` — the npm migration audit.
- `CHANGELOG.md` — what's shipped, entry by entry.
