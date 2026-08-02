# Releasing guide

How to ship a new version of Lumo — the GitHub release and the npm
package — and how to recover when something goes wrong. The process is
automated but *owned*: the tag is the trigger, and the checks below are
what make a release repeatable rather than a one-off ritual.

## How a release works

Pushing a semver tag `vX.Y.Z` on `main` runs
[`.github/workflows/release.yml`](../../.github/workflows/release.yml):

1. **`release` job** — cross-compiles the 5-platform release matrix
   (ADR-0006), stages the V1 plugin set into each binary via go:embed
   (ADR-0012), builds the archives, writes `SHA256SUMS.txt`, and creates
   the GitHub Release with generated notes.
2. **`publish-npm` job** (after the release exists — the npm installer
   downloads those assets, so ordering is correctness):
   - fails unless `distribution/npm/package.json`'s version equals the
     tag;
   - re-verifies every release asset via `prepublishOnly`
     (`scripts/verify-release.js`);
   - `npm publish --provenance`;
   - smoke-tests the packed tarball on a clean prefix
     (`lumo version` + `lumo plugins list`).
   - The job is skipped (not failed) if the `NPM_TOKEN` secret is not
     configured — the GitHub release still ships.

Other automation on a tag: none. `ci.yml` and `distribution-verify.yml`
run on PRs and `main`, not on tags, so a release run exercises the
pipeline, not the tests (the tests ran before the tag was cut).

## Before you cut a release

- [ ] `CHANGELOG.md` has an `## [x.y.z]` section moved from
      `[Unreleased]` (Keep a Changelog, ADR-0006). The changelog
      **is** the release notes.
- [ ] If this release changes the npm wrapper (it usually doesn't —
      the wrapper changes only when the release does), `bump
      distribution/npm/package.json`'s version **in the same commit**
      as the changelog entry. The wrapper version must equal the tag
      exactly, or `publish-npm` fails.
- [ ] Breaking changes to a Stable surface have their migration-guide
      entry (`docs/guides/migration-guide.md`) and an ADR — see
      `docs/guides/versioning.md`.
- [ ] `main` is green (CI + distribution-verify), and `git status` is
      clean — a `--dirty` build stamps a dirty version string into the
      binaries.
- [ ] Plugin authors: `lumo plugins validate` still passes on the
      V1 set (it runs in CI, but the local check catches it before the
      tag, not after).

## Cut the release

```sh
git checkout main && git pull
git tag -a vX.Y.Z -m "vX.Y.Z"
git push origin vX.Y.Z
```

Watch the run: the release job creates the GitHub Release; if
`NPM_TOKEN` is configured, `publish-npm` publishes and smoke-tests the
npm package. A failed publish **fails the run visibly** — fix the
cause, bump the version forward (never reuse a failed tag), and tag
again.

## Verify after the release

- GitHub Release page: 5 archives + `SHA256SUMS.txt`, correct title.
- npm: `npm view lumo-cli version` shows the new version, and
  `npm view lumo-cli dist-tags` shows it as `latest`.
- Clean-machine check on the registry package (both major platforms):
  run `.github/workflows/npm-verify-published.yml` manually
  (workflow_dispatch). It installs `lumo-cli@latest` on a
  disposable Linux and Windows runner and runs `lumo version` and
  `lumo plugins list`.

## First-ever npm publish (done 2026-08-02)

The npm channel is live as the interim package
**`bootstrap-cli-dev@0.3.0`** — published under the platform's old
name, before the rename. `lumo-cli` (version-synced with the `lumo`
release it launches, per ADR-0015) is the successor name and will be
published as `lumo-cli@1.0.0` with the v1.0.0 release, replacing the
interim package (deprecated 2026-08-02 — see
`distribution/npm/README.md`'s Status section). What the first publish
took, as a record for the remaining ecosystems:

1. Created an npm **automation token** (npmjs.com → Access Tokens →
   Generate New Token → *Automation* — automation tokens bypass the
   OTP prompt; they do **not** waive the account-level 2FA that npm
   requires of every publisher).
2. Added it as the repository secret **`NPM_TOKEN`** (Settings →
   Secrets and variables → Actions). The `publish-npm` job now runs
   on every release tag.
3. Published `v0.3.0` directly from a maintainer machine (the release
   pipeline wasn't wired yet at that point), then re-verified the
   published package with a clean `npm install bootstrap-cli-dev` —
   `lumo version` and `lumo plugins list` both passed.

Notes for future publishers: provenance signing
(`npm publish --provenance`, the workflow's explicit flag) needs
GitHub Actions OIDC — it cannot be generated from a local publish.
If a future publish fails with an OIDC error, check the repo's
Actions → General OIDC setting; provenance is deliberately not in
`publishConfig` so local publishes don't fail on it.

## Rolling back a bad npm release

npm versions are immutable — you cannot re-publish over one, and
`npm unpublish` is time-limited (72 hours) and leaves a scar. The
supported recovery is **deprecate + forward-fix**, in that order:

1. **Deprecate** so no one new installs it:
   `npm deprecate lumo-cli@X.Y.Z "broken: <cause> — upgrade to X.Y.Z"`.
   Deprecation messages show on every install of that version.
2. **Fix and release forward**: the GitHub release for the bad version
   already exists (the npm wrapper downloads *its* assets, which are
   immutable), so ship the fix as the next tag `vX.Y.Z+1` (or a minor
   bump), which republishes the npm package at the new version. The
   GitHub release for the bad version is historical record; add a note
   to its notes pointing at the fixed release.

Never delete a GitHub release that an npm version points at — the npm
package's postinstall would download nothing, and every installed copy
of that version would be permanently broken.

## Maintenance (the 5-year view)

- **Release cadence.** The npm package costs nothing extra per
  release: it republishes automatically with each tag. There is no
  separate npm maintenance.
- **Keep the wrapper boring.** The wrapper must not grow features. If a
  fix needs touching `distribution/npm/scripts/*`, it ships with the
  next release tag — bumping the wrapper without a release would
  violate version-sync. Wrapper-only changes are acceptable *only* as
  an emergency patch and must bump the wrapper version to the same
  version as the *next* release; prefer folding them into the next
  tag.
- **Re-verify the asset matrix quarterly** (or when `release.yml`'s
  `TARGETS` changes): `scripts/verify-release.js` and
  `distribution-verify.yml`'s npm job cover every supported platform;
  if a target is added upstream, extend all three places
  (`TARGETS`, `platformTarget()`/`verify-release.js` targets,
  `os`/`cpu` in `package.json`) in the same change.
- **Watch the ecosystem.** npm changing install-script policy would
  affect every download-at-install CLI (ADR-0015 records the trigger to
  reconsider per-platform optional packages). GitHub changing release
  asset URLs or checksums would affect the postinstall directly —
  the `distribution-verify.yml` npm job catches this the moment it
  happens.
- **Rotate the token.** Automation tokens don't expire by default; set
  a reminder to rotate `NPM_TOKEN` annually and re-run
  `npm-verify-published.yml` after rotating.
