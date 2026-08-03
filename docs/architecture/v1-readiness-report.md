# Lumo — roadmap to v1.0.0, readiness audit, and release plan

> Principal-architect review, 2026-08-02. Current release: **v0.3.0** (tag
> `v0.3.0`, GitHub "Latest"). Target: a **stable, trusted v1.0.0**.
> Companion to `docs/ai-context-report.md` (full project state) and the
> existing `docs/architecture/roadmap.md` (non-goals).

This document delivers, in order:

1. **Version-by-version roadmap** from v0.3 → v1 (incremental, focused per release).
2. **Must-do checklist before v1** (the non-negotiable bar to call something v1).
3. **Release checklist** (the mechanical steps to cut, sign, publish, verify).
4. **Risk report** (scored, with mitigations).
5. **Final recommendation** — ready / release-candidate / not-ready, with reasons.

Guiding principle: **trust, reliability, maintainability over speed.** No
feature ships to hit a date; every release narrows uncertainty.

---

## 1. Version-by-version roadmap (v0.3 → v1.0.0)

Five planned releases between today and a stable v1.0.0. Each is small enough
to cut in days-to-a-week, each removes one class of risk, each is independently
verifiable. The semver contract (§3) is published at v0.4.0 and honored
thereafter.

```text
v0.3.0  ──►  v0.4.0  ──►  v0.5.0  ──►  v0.6.0  ──►  v0.7.0  ──►  v0.8.0 (RC) ──►  v1.0.0
current    foundation   installer    audit+      plugin      release
           + semver     + refactor   hardening   ecosystem   candidate
```

### v0.4.0 — Foundation, semver contract, docs truth

**Theme: make the ground truth true.** Today several docs and manifests describe
a state that doesn't exist yet (the most consequential: the distribution
wrappers expect `lumo_v1.0.0_*` assets that no release provides; status docs
claim publish-readiness; `go install` is described as unsupported without an
alternative). v0.4.0 stops the drift.

Scope:

- **Publish the semver & stability policy** (`docs/guides/versioning.md` is
  already drafted; tighten it and make it normative). Define which surfaces
  are Stable at v1 (wire protocol, `sdk/go`, CLI commands, distribution
  contract — per ADR-0013) and the Experimental→Stable→Deprecated→Removed
  lifecycle. State the breaking-change bar (a minor bump minimum; a migration
  guide entry; an ADR for protocol/wire changes).
- **Audit + repair all stale claims** across docs and manifests (the corrupt
  prose is already fixed on `main`; sweep for residual overclaims — e.g.
  `distribution-verify.yml` cannot currently pass, status tables must reflect
  "pending v1.0.0 release", `go install` guidance).
- **Reconcile the wrapper asset naming.** Decision: keep wrappers pointed at the
  upcoming `lumo_v1.0.0_*` names and mark `distribution-verify.yml` as
  post-release-only (already true by trigger; document it), **or** temporarily
  repoint them at the existing `cli_v0.3.0_*` assets so CI is green today. The
  first is cleaner; pick it and add a workflow-status note.
- **Finalize the docs suite**: install, quick start, API reference, contribution,
  versioning. (Most exist; this is a polish + correctness pass, not new
  scaffolding.)
- **Code of conduct + security policy** (`SECURITY.md`) published.
- **Markdownlint re-run** over the whole `docs/` tree; fix all violations.

Exit criteria: every doc statement is true as of the commit it lands on; the
semver policy is referenced from `README.md`; `distribution-verify.yml`'s
expected state is documented; no lint violations.

### v0.5.0 — Installer design: cross-platform, PATH, safe upgrade, rollback

**Theme: installation is the first trust signal.** A scaffolding tool that
can't be installed cleanly, upgraded safely, or rolled back is not v1-worthy.
The current installers (`install.sh`, `install.ps1`, the npm postinstall) work
but lack a coherent, tested contract.

Scope (design + implementation, ADR-gated):

- **New ADR: installer architecture.** Defines the install contract for every
  channel: where the binary lives (`~/.lumo/bin` / `%USERPROFILE%\.lumo\bin`
  for the script installers; package-manager default for those), how PATH is
  managed per-OS (macOS/Linux shell-rc editing with user consent + dry-run;
  Windows `User` `Path` env var via `setx`/registry, never `Machine`), how the
  embedded plugin fallback (ADR-0012) makes the binary self-sufficient so the
  install is a single file plus a versioned cache dir.
- **Safe upgrade + rollback.** Install into
  `os.UserCacheDir()/lumo/<version>/` (already ADR-0012's cache layout); a
  `lumo` shim/symlink in `~/.lumo/bin` points at the active version. Upgrading =
  install new + repoint + verify (`lumo version`); the previous version stays
  on disk. `lumo upgrade --rollback` repoints back. npm/channel upgrades stay
  idempotent via the existing `.lumo-version` marker.
- **`lumo install --dry-run`** prints exactly what would change (paths, PATH
  edits, version) without touching the system.
- **Checksum verification** on every channel (already in npm postinstall; extend
  to `install.sh`/`install.ps1` against `SHA256SUMS.txt`, with a clear failure
  on mismatch).
- **Uninstall** (`lumo uninstall`) that reverses PATH edits and offers cache
  cleanup.
- **Tests**: an install/upgrade/rollback integration suite (script-driven, per
  OS, in CI where feasible — at minimum a containerized Linux + a Windows
  runner; macOS remains manual-or-act).

Exit criteria: the installer contract is documented; install/upgrade/rollback
is tested on at least Linux + Windows; dry-run is honest; checksum failures
are loud.

### v0.6.0 — Architecture + security audit, refactor, error & log & test hardening

**Theme: prove the inside is as clean as the outside.** This is the
no-new-features release. It earns the "stable" in v1.0.0.

Scope:

- **Full architecture audit**: walk every module boundary (`cli` → `core` →
  `sdk/go` → plugins/templates); confirm imports match the table in
  `overview.md`; remove any accidental coupling; confirm `go.work` cleanliness.
  Output: an audit doc under `docs/architecture/`.
- **Security audit** (threat model + review): subprocess spawning of plugin
  binaries (PATH/`plugin.json` trust boundary, manifest tampering, working-dir
  resolution, `LUMO_PLUGIN_DIRS` injection), the npm postinstall download
  (`https`-only, checksum verify, redirect handling, TOFU note), the wrapper
  `bin/lumo.js` spawn (`stdio: inherit`, argv forwarding), secret handling
  (`NPM_TOKEN`, never logged), the GitHub-release asset trust chain (provenance
  with checksums). Findings → `docs/architecture/security-audit.md` with severity,
  exploitation preconditions, and fixes. (Expected class: plugin execution is
  inherently "you run code you installed" — document the boundary, don't pretend
  to sandbox it; focus hardening on *transport* integrity and *manifest*
  authenticity.)
- **Error handling pass**: every user-facing error is actionable (names the
  failing step, suggests the fix, points at docs/`lumo doctor`); exit codes are
  consistent (0/1/2/130). Replace bare `fmt.Errorf` user paths with structured
  errors where it helps `--verbose`.
- **Logging**: a single `--verbose` discipline (no ad-hoc `fmt.Println`); a
  `LUMO_DEBUG` escape hatch for maintainer dumps; ensure **no secrets or
  user-private paths** leak into logs.
- **Test coverage**: bring unit coverage on `core/` and `cli/internal/prompt`
  to an explicit target (propose ≥80% line coverage on `core/{engine,plugin,
  registry,config}`); add property/table tests for the wizard's filtering logic;
  add negative tests (corrupt manifest, bad checksum, missing binary, broken
  pipe, Ctrl+C at every stage).
- **Refactor where the audit demands it** — no cosmetic rewrites; only
  correctness/clarity wins.

Exit criteria: audit docs accepted by self-review with a recorded sign-off;
coverage target met; zero `// TODO`/`FIXME` that map to a v1 risk; security
audit's Criticals and Highs all fixed or explicitly accepted with an ADR.

### v0.7.0 — Plugin ecosystem: discovery, validation, third-party path

**Theme: the platform's reason to exist.** With the core hardened, make adding
a stack a one-plugin operation that a third party can actually do safely.

Scope:

- **`lumo plugins` UX**: `list` (today), `validate <dir>` (today), plus
  `lumo plugins search <query>` over a *local* index (a remote registry remains
  a non-goal for v1 — see `roadmap.md`; but a documented local-index format +
  a `lumo plugins add <path-or-git-url>` that clones/installs into a
  user-writable plugin dir and runs `validate` is in-scope and useful).
- **Third-party plugin path proven**: a documented, tested workflow for an
  external author to write, build, ship, and have Lumo discover a plugin —
  across Go and (via the language-neutral SDK spec, ADR-0014) at least one
  non-Go language as a proof (`sdk/go` is the reference; ship `sdk/node` or
  `sdk/python` as the cross-language proof, CI-verified clean-machine round-trip,
  same bar as wrappers).
- **Plugin sandboxing boundary documented** (not implemented): a clear
  `SECURITY.md`-level statement that plugins run as subprocesses with full
  process privileges — installation is consent — plus the integrity guarantees
  we *do* provide (manifest validation, protocol handshake, checksummed
  distribution).
- **Capability conflict detection** (deferred in `roadmap.md`): a *minimal*
  version — detect two capabilities writing the same path and fail loudly
  (ordering stays; no merge resolution). Closes a real footgun cheaply.

Exit criteria: a non-maintainer can author and install a plugin following only
docs (no tribal knowledge); one non-Go SDK ships round-trip-verified; the
conflict-detection footgun is closed.

### v0.8.0 — Release candidate (RC)

**Theme: freeze and prove.** v0.8.0 is tagged `v1.0.0-rc.1` on the registry
channel (npm `lumo-cli@1.0.0-rc.1`, GitHub `v1.0.0-rc.1`). Everything that will
be in v1.0.0 is here; the only changes after this are fixes from RC feedback.

Scope:

- Cut the **real v1.0.0 GitHub release assets** (`lumo_v1.0.0_*` +
  `SHA256SUMS.txt`) via `release.yml` — this unblocks the wrappers and the npm
  publish.
- Publish **`lumo-cli@1.0.0-rc.1`** on npm with provenance (the pre-publish
  guard now passes).
- **Cross-platform clean-install matrix**: macOS (amd64 + arm64), Windows
  (amd64), Linux (amd64 + arm64), all via the npm wrapper *and* the
  `install.sh`/`install.ps1` channels; each runs `lumo version`, `lumo doctor`,
  `lumo new` (interactive and `--answers`), `lumo plugins list`, and a generated
  project's own test suite.
- **Signed artifacts**: provenance (npm) + cosign/Sigstore signing of release
  archives if feasible in the release pipeline; minimally, document the
  `SHA256SUMS.txt` verification and attach a maintainer PGP signature.
- **Release notes** drafted (user-facing, not commit-log): what Lumo is, how to
  install, what's stable, what's experimental, known limitations, migration from
  `bootstrap-cli-dev@0.3.0`.
- **Two-week soak**: RC live, advertised in README/issue tracker; collect
  install reports; no Critical/High findings open before promoting to v1.0.0.

Exit criteria: clean installs confirmed on all 5 platform targets via 2+
channels; `npm-verify-published.yml` green; release notes reviewed; no open
Critical/High.

### v1.0.0 — Stable

**Theme: the promise.** Same bits as the RC if nothing regressed; the promotion
is a version tag, a promoted npm `latest`, deprecation-finalization of the
interim package, and the announcement.

Scope:

- Tag `v1.0.0`; `release.yml` produces assets + publishes `lumo-cli@1.0.0`.
- Promote npm `lumo-cli@1.0.0` to `latest`; finalize the
  `bootstrap-cli-dev@0.3.0` deprecation (already deprecated 2026-08-02).
- Flip all status docs to "published".
- Publish `docs/guides/migration-guide.md` entry for v1.0.0; freeze the Stable
  surfaces list.
- Announcement (GitHub release body + README banner).

### Post-v1 (not in this roadmap's scope, sequenced for after)

- **CLI UX refresh** — best-in-class polish with Lumo's own identity (informed
  by RC soak feedback), shipped as v1.1.x.
- **Other package managers, one at a time** — Homebrew, Scoop, Winget,
  Chocolatey, PyPI, Cargo — each its own PR-with-tests, each gated on a real
  credential/PR; v1.2.x+, one minor bump per addition so each is independently
  revertable.
- **Plugin marketplace + website** — only after the local plugin path (v0.7.0)
  has real-world mileage; a remote registry is a v2-scale effort and needs its
  own ADR + trust model.

---

## 2. Must-do checklist before v1

The bar to call a release "v1.0.0". None of these is optional; each maps to a
release above.

### Architecture & code

- [ ] Full module-boundary audit passes (no forbidden imports; `go.work`
      clean). `docs/architecture/` audit doc signed off.
- [ ] No `// TODO`/`// FIXME` that correspond to a tracked v1 risk.
- [ ] Every public surface (ADR-0013) is tagged Stable or Experimental in code
      docs (`sdk/go` package doc already done; extend to `core`'s exported APIs).
- [ ] Error handling: every user-facing error actionable; exit codes 0/1/2/130
      consistent across paths.
- [ ] Logging: single `--verbose` discipline; `LUMO_DEBUG`; no secret/private
      path leakage.

### Security

- [ ] Threat model written (`docs/architecture/security-audit.md`): plugin
      execution boundary, transport integrity, manifest authenticity, secret
      handling, asset provenance.
- [ ] All Critical/High findings fixed or ADR-accepted; remaining Mediums
      documented with a CVE-style entry each.
- [ ] `SECURITY.md` published (reporting policy, supported versions, SLA).
- [ ] Release artifacts signed (provenance at minimum; cosign if feasible);
      `SHA256SUMS.txt` verification documented and tested.
- [ ] No hardcoded secrets in tree; `NPM_TOKEN`/automation tokens validated as
      never printed.

### Tests

- [ ] Unit coverage ≥80% on `core/{engine,plugin,registry,config}`.
- [ ] Negative tests: corrupt manifest, bad checksum, missing binary, broken
      pipe, Ctrl-C at each stage, unsupported platform (`EBADPLATFORM`).
- [ ] Wire-protocol golden transcripts green; `-update` produces no diff.
- [ ] Integration: bare-wizard, `--answers`, exit codes, `plugins validate`,
      wizard piped fallback, no-templates fail-fast — all green.
- [ ] Cross-platform clean-install matrix green (macOS×2, Linux×2, Windows×1)
      on ≥2 channels.

### Installer

- [ ] Install contract ADR'd; install/upgrade/rollback/uninstall implemented;
      dry-run honest; PATH edits consented + reversible.
- [ ] Checksum verification on every channel; failures loud.
- [ ] Idempotent re-install (npm `.lumo-version` marker; script installer
      equivalent).

### Distribution & release

- [ ] `lumo_v1.0.0_*` archives + `SHA256SUMS.txt` produced by `release.yml`.
- [ ] `verify-release.js` passes (all 6 assets 200).
- [ ] `lumo-cli@1.0.0` published with provenance; `npm view lumo-cli version`
      returns `1.0.0`; `latest` dist-tag points at it.
- [ ] `bootstrap-cli-dev@0.3.0` deprecation finalized with a pointer to
      `lumo-cli@1.0.0`.
- [ ] `npm-verify-published.yml` green on Linux + Windows.

### Docs & policy

- [ ] Semver & stability policy normative and referenced from `README.md`.
- [ ] Install, quick start, API reference, contribution, versioning, migration
      guide all current and true.
- [ ] `CHANGELOG.md` v1.0.0 entry complete (user-facing prose).
- [ ] `roadmap.md` non-goals list reconciled (deferred items still deferred by
      design, not by accident).

### Semver guarantees to declare at v1

- **Stable (semver-bound):** the JSON-RPC wire protocol (`protocolVersion`), the
  `sdk/go` public API, the CLI command & flag surface, the distribution wrapper
  contract, the `plugin.json` manifest schema version.
- **Experimental (no semver promise):** the `cli/internal/prompt` theme
  registry, the `core/*` package internals, anything under `sdk/{node,python,
  rust,future}` design notes.
- **Breaking changes** require: a minor bump (or major, per semver), a
  `migration-guide.md` entry, and — for wire/protocol — an ADR. No silent
  breaking changes; old plugins keep working across a protocol bump until
  multi-version negotiation is built (deferred).

---

## 3. Release checklist (v1.0.0, and reusable for each v0.x cut)

Reusable per-release; the v1.0.0 specifics are annotated.

### Pre-cut (days before)

- [ ] All "must-do before v1" items green for v1.0.0 (or the subset scoped for
      this v0.x release).
- [ ] `CHANGELOG.md` entry written (Keep a Changelog; Added/Changed/Deprecated/
      Fixed).
- [ ] Version bumped in `distribution/npm/package.json` (must equal the tag —
      `release.yml` enforces).
- [ ] `docs/guides/migration-guide.md` entry added for any breaking change.
- [ ] Full local test run green: `go test ./...` (per module) + integration
      `go test -count=1 ./tests/...`.
- [ ] `gofmt -l` reviewed (pre-existing untouched files noted, not new ones
      introduced).
- [ ] `markdownlint` green over touched docs.
- [ ] `distribution-verify.yml` expected state confirmed (green for v0.x if
      wrappers repointed at a real release; red-but-documented for the
      pre-v1.0.0 window — see risk R-03).

### Cut

- [ ] Tag `vX.Y.Z` from `main`; push the tag.
- [ ] `release.yml` runs: archives `lumo_vX.Y.Z_*` + `SHA256SUMS.txt` created;
      GitHub Release published.
- [ ] (v1.0.0 only) `publish-npm` job runs: version-sync check passes,
      `prepublishOnly` (`verify-release.js`) passes, `lumo-cli@X.Y.Z` published
      with provenance.

### Post-cut (same day)

- [ ] `npm view lumo-cli version` → `X.Y.Z`; `npm view lumo-cli dist-tags` →
      `latest` = `X.Y.Z`.
- [ ] `npm-verify-published.yml` green (Linux + Windows clean-machine install +
      `lumo version` + `lumo plugins list`).
- [ ] Manual cross-platform smoke: install via `install.sh`/`install.ps1` on
      one Linux + one macOS + one Windows; `lumo new` interactive; `lumo new
      --answers`; generated project builds + tests pass.
- [ ] (v1.0.0 only) Finalize `bootstrap-cli-dev@0.3.0` deprecation; verify the
      registry message points at `lumo-cli@1.0.0`.
- [ ] (v1.0.0 only) Flip status docs to "published": `distribution/npm/README.md`,
      `distribution/README.md`, `roadmap.md`, `releasing.md`.
- [ ] Update the GitHub Release body with user-facing release notes.
- [ ] Announcement (README banner / issue / Discord if any).

### Rollback (if anything goes wrong)

- [ ] `npm deprecate lumo-cli@X.Y.Z "<cause> — pin to X.Y.(Z-1)"`.
- [ ] If the release assets themselves are bad: delete the tag's GitHub Release
      (assets), keep the tag as a record, cut `X.Y.(Z+1)` forward (never reuse a
      tag), repoint wrappers, re-publish.
- [ ] Postmortem entry in `CHANGELOG.md` next release's `Fixed`.

### Signing (v1.0.0)

- [ ] npm provenance verified on the published package (`npm view
      lumo-cli@1.0.0 --json` shows provenance).
- [ ] GitHub Release `SHA256SUMS.txt` attached; a maintainer PGP/cosign signature
      of `SHA256SUMS.txt` attached if sigstore is wired; verification command
      documented in `docs/guides/install` or `releasing.md`.

---

## 4. Risk report

Severity: **C**ritical (blocks v1) · **H**igh (must fix before v1) ·
**M**edium (acceptable for v1 with mitigation) · **L**ow (track).

| ID | Risk | Sev | Likelihood | Impact | State / mitigation |
|---|---|---|---|---|---|
| R-01 | **Distribution wrappers point at non-existent assets.** The 6 non-npm wrappers expect `lumo_v1.0.0_*` at the `v1.0.0` tag; the latest release is v0.3.0 with `cli_v0.3.0_*` assets (verified 2026-08-02: `lumo_v1.0.0_windows_amd64.zip` → 404; `cli_v0.3.0_*` → 200). `distribution-verify.yml` would fail on every wrapper job today. | H | Resolved 2026-08-03 | — | **Resolved at v0.4.0.** The v0.4.0 release cut real `lumo_v0.4.0_*` assets; all 6 non-npm wrappers were repinned to them (URLs + real checksums from the release `SHA256SUMS.txt`, winget installer sha256, CI path) and `distribution-verify.yml` runs green on all 7 jobs (verified 2026-08-03). Wrappers are now repinned as part of each release cut; the workflow's expected-state comment enforces this. |
| R-02 | **`bootstrap-cli-dev@0.3.0` is the only live npm package, and it's deprecated.** Until `lumo-cli@1.0.0` publishes, the deprecation warning points users at a package that doesn't exist yet. | M | Resolved 2026-08-03 | — | **Resolved at v0.4.0.** `lumo-cli@0.4.0` (renamed, version-synced wrapper) was published to npm by `release.yml` on the v0.4.0 tag with provenance, and its tarball smoke test passed on a clean prefix. `bootstrap-cli-dev` remains deprecated and continues to work; users are pointed at the now-live `lumo-cli`. |
| R-03 | **`go install` is unsupported** (no embedded fallback; needs sibling dirs; no `cli/vX.Y.Z` submodule tags). README documents it as "not yet recommended" but offers no path. | M | Conditional (a Go dev tries `go install`) | Confusion, silent broken binary | v0.4.0 either softens the wording or ships `cli/v0.4.0` submodule tags so `go install@v0.4.0` works. v0.5.0's installer work supersedes for non-Go channels. |
| R-04 | **Plugin execution is a trust boundary we don't sandbox.** Any installed plugin runs as a full subprocess (filesystem, network, env). | M | Inherent to the design | Malicious plugin = arbitrary code under user's account | Not fixable for v1 without redesign; v0.6.0 security audit must *document* it as user-consent at install time + provide integrity guarantees (manifest validation, checksummed distribution, protocol handshake). Acceptable with `SECURITY.md` statement. |
| R-05 | **Wire protocol has no multi-version negotiation.** A protocol bump drops every old plugin simultaneously. | M | Low (no bump planned for v1) | An ecosystem break if a bump is needed | Out of v1 scope (deferred, `roadmap.md`). Mitigation: declare `protocolVersion` Stable; no bump before v1.1 unless forced, and then with an ADR + migration guide. |
| R-06 | **No capability conflict detection.** Two capabilities touching the same file produce undefined output (last-writer-wins, silent). | M | Low-moderate | Surprising generated projects | v0.7.0 ships minimal detection (fail loudly on path overlap). Cheap, closes the footgun. |
| R-07 | **Test coverage unmeasured.** The suite is green but per-package coverage isn't tracked; "green" could be shallow. | M | Possible | Untested paths ship broken in v1 | v0.6.0 sets an ≥80% target on `core/*`, adds negative tests, and wires coverage into CI. |
| R-08 | **Pre-existing `gofmt`-unformatted files** (`config.go`, `config_test.go`, `theme_test.go`) left untouched by prior phases. | L | Real | Distracts reviewers; CI `gofmt -l` noise; risk of accidental reformatting later | v0.4.0 sweeps them once (deliberate, single commit) so the tree is clean going forward. |
| R-09 | **Direct push to `main` (commit `872c962`) broke the PR-first convention.** Sets a precedent; future contributors might push directly too. | L | Possible | Review bypass; harder revert; bisect harder | v0.4.0 reinforces the convention in `CONTRIBUTING.md`; going forward every release-relevant commit goes through a PR (even if merged immediately).tracked. |
| R-10 | **Cryptographic signing is npm-provenance-only.** Release archives rely on `SHA256SUMS.txt` transport over GitHub's TLS; no detached signature. | L | Possible | A compromised GitHub asset could be undetectable without a second trust anchor | v0.8.0 evaluates cosign/Sigstore for archives; at minimum attaches a maintainer PGP signature to `SHA256SUMS.txt`. Documented in `SECURITY.md`. |
| R-11 | **No_remote plugin registry (intentional).** Plugin discovery is local-only. Some users will expect `lumo plugins search <x>` against a marketplace. | L | Expected | Mild disappointment; not a correctness risk | Declared non-goal in `roadmap.md`; v0.7.0's local-index `search` softens the UX. Remote registry is a v2 effort. |
| R-12 | **Single-maintainer bus factor.** The npm account, GitHub secrets, and release-cutting knowledge live with one person. | M | (org reality) | Release pipeline stalls if the maintainer is unavailable | v0.4.0+: document everything (this is largely done); ensure `releasing.md` lets a second maintainer cut a release; consider a second npm owner (`npm owner add`) at v1.0.0. |

**Aggregate judgement:** no **Critical** risks open. The **High** (R-01) is
resolved — the v0.4.0 release cut real assets and all wrappers were
repinned to them, with `distribution-verify.yml` green. The **Mediums** are either
inherent-and-acceptable (R-04, R-05), cheaply closed within the roadmap (R-06,
R-07), partially resolved (R-02, R-03), or transitory. None blocks proceeding past v0.4.0.

---

## 5. Final recommendation

### Verdict: **Release-candidate track — not ready for v1.0.0 today, but on a credible path.**

Lumo is **not ready to ship as v1.0.0 today**, for three concrete reasons:

1. **The distribution layer was mid-transition.** As of v0.4.0 (2026-08-03)
   this is now resolved: the release cut real assets, all wrappers are
   repinned and `distribution-verify.yml` is green (R-01), and the renamed
   `lumo-cli@0.4.0` npm package is live (R-02). The remaining
   release-readiness work is the v0.6.0 audit and the v0.8.0 RC — cutting
   v1.0.0 today would still skip those gates.

2. **The trust documentation isn't fully normative yet.** The semver/stability
   policy exists in draft but hasn't been declared binding from `README.md`;
   `SECURITY.md` isn't published; the security audit and architecture audit
   haven't been run (v0.6.0). Shipping v1.0.0 *means* "this surface is stable
   and here are the guarantees" — that promise isn't ready to be made.

3. **Test coverage is unmeasured and the installer contract is unproven.**
   Green tests with unknown depth, plus an installer that works but hasn't
   been hardened for safe upgrade/rollback across platforms, is below the bar
   for a release that explicitly asks for long-term trust.

### What "ready for v1.0.0" looks like (the path this roadmap lays out)

- **v0.4.0** makes the docs true and the policy binding.
- **v0.5.0** proves install/upgrade/rollback.
- **v0.6.0** runs the audits and hardens errors/logs/tests.
- **v0.7.0** proves the platform is extensible by a third party.
- **v0.8.0** cuts the RC, the real v1.0.0 assets, and the npm RC publish, and
  soaks for two weeks.
- **v1.0.0** promotes the RC: tag, npm `latest`, finalized deprecation, status
  flips, announcement.

### Trust priorities reaffirmed

- **Reliability over speed**: the v0.4→v0.8 cadence exists to *remove* risk
  each step, not to hit a date. If v0.6.0's audit finds a Critical, the roadmap
  pauses there.
- **Maintainability**: every audit produces a doc; every release produces
  release notes and a migration entry; the PR-first convention is restored.
- **Trust**: `SECURITY.md`, signed artifacts, honest installers, and a semver
  promise that names *exactly* what's stable and what's not.

### One-line summary for the maintainer

> Lumo has the architecture, UX, and a working npm interim channel; it does
> not yet have the audited core, normative trust docs, proven installer, or
> the real v1.0.0 assets that a trustworthy v1.0.0 requires. Execute the
> v0.4→v0.8 roadmap, cut an RC, then promote.
