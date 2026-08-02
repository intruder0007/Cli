# ADR-0015: First published wrapper — npm package identity and release coupling

## Status

Accepted

## Context

ADR-0010 defined a distribution contract and `distribution/` now contains
seven built, CI-verified wrappers — but none are published to a live
registry, and every wrapper's own README says so. Publishing is the next
step the roadmap names, and npm is the lowest-friction candidate: the
wrapper is complete, verified against real release assets in
`distribution-verify.yml`, and the ecosystem has no third-party review
gate (unlike Homebrew/Scoop/Winget PRs).

Publishing surfaces two decisions that the wrapper design deliberately
did not settle:

1. **Package name.** The working name `bootstrap-cli` is already taken
   on npm (a different project). This is a permanent, hard-to-reverse
   identity decision — the name is what users type, search, and file
   issues against for years.
2. **How the wrapper's version tracks the `bootstrap` release it
   launches.** The wrapper contract (distribution-protocol.md) says the
   wrapper version "should track (not necessarily equal, but traceable
   to)" the release. Left unpinned, that gives exactly one release a
   year a real chance to publish a wrapper whose version says one thing
   and whose binary is another — the failure the whole distribution
   contract exists to prevent.

Plus one structural question: the npm wrapper downloads the release
archives at install time (postinstall script), so publish ordering —
GitHub release first, npm publish second — is not a preference but a
correctness requirement.

## Decision

**Publish the npm wrapper as `bootstrap-cli-dev`, with the wrapper
version *always equal* to the `bootstrap` release it launches, and
couple npm publishing to the release pipeline itself.**

- **Name: `bootstrap-cli-dev`.** Unscoped, un-taken (checked at
  decision time), one-word memorable, and self-describing: this is the
  dev-tooling entry point for the platform. A scoped name
  (`@intruder0007/cli`) was the runner-up: unambiguous ownership but
  weaker as a public product identity and it reads as personal, not
  product. No `-dev` semantics beyond the name — the package is a
  production distribution channel, not a nightly/dev tag.
- **Wrapper version = release version, enforced, not advised.** The
  version-sync guard in `release.yml`'s `publish-npm` job fails the job
  (publishing nothing) unless
  `distribution/npm/package.json`'s version equals the git tag. The
  postinstall idempotency marker (`distribution/npm/.bin/.bootstrap-version`)
  is keyed on the package version, and the pre-publish guard
  (`scripts/verify-release.js`, run by `prepublishOnly` and CI) checks
  that *every* platform archive + `SHA256SUMS.txt` exists for the
  package's version before any publish succeeds. Three independent
  checks bind wrapper version to release version; no check, no publish.
- **Publish inside the release pipeline, after the GitHub release.**
  `release.yml` gains a `publish-npm` job that runs only after the
  `release` job has created the release and its assets — the assets the
  npm postinstall downloads, so the ordering guarantees an installable
  package. The job is gated on the `NPM_TOKEN` repository secret
  existing; until a maintainer adds it, tags create GitHub releases and
  npm publishing is skipped (documented, intentional, and visible in
  every release's check run).
- **Keep the download-at-install pattern.** The protocol (step 2) allows
  either per-platform optional-dependency packages or a postinstall
  download. The existing wrapper already implements the download +
  checksum-verify path, verified end-to-end; switching to five
  platform-variant packages multiplies publish surface (six packages to
  keep in sync per release) without fixing a real failure. The one
  inherent cost — install scripts need network and can be disabled via
  `--ignore-scripts` — is surfaced by the shim with an actionable error
  and documented. Revisit only if install-script restrictions become
  widespread on npm.

## Consequences

- `distribution/npm/package.json` is production metadata: name,
  description, author, repository/bugs/homepage, keywords, `os`/`cpu`
  restrictions (matches the release matrix exactly — notably no
  `windows/arm64`, which is not built, so such installs fail with a
  clear `EBADPLATFORM` instead of a broken binary), and `engines`
  (Node ≥ 18). Provenance signing is enabled via the workflow's
  explicit `npm publish --provenance` flag (GitHub OIDC), not
  `publishConfig` — npm cannot generate provenance outside CI, and a
  baked-in `publishConfig.provenance` would fail local publishes with
  an opaque error.
- Every release tag now *also* publishes the npm wrapper (once
  `NPM_TOKEN` exists). A failed npm publish fails the tag's run visibly
  rather than half-releasing; a skipped npm publish (no token yet) is
  explicit in the check summary.
- The npm package does not require a separate release cadence: bump the
  wrapper version in the same commit as the release's changelog entry.
- `EBADPLATFORM` (e.g. FreeBSD, or arm64 Windows) is the designed
  failure mode — unsupported targets get npm's standard error rather
  than a postinstall that downloads nothing and a shim that mysteriously
  fails.
- Verified before acceptance: local `npm pack` + install into a clean
  scratch prefix on Windows — download, checksum verify, extraction,
  `bootstrap version`, and `bootstrap plugins list` (all five embedded
  plugins) all passed through the real npm shim; the marker short-circuits
  re-downloads; and the missing-binary shim error is actionable. The
  same smoke test runs in CI for every publish.
