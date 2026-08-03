# `lumo-cli` — the npm wrapper

The npm distribution channel for the Lumo project scaffolding platform.
Install the real `lumo` binary for your platform, verify it, and
run it — through a thin launcher that never reimplements the CLI
(see the [wrapper protocol](../../docs/architecture/distribution-protocol.md)).

## Install

```sh
npm install -g lumo-cli
# or, per-project (non-global):
npm install --save-dev lumo-cli
```

Then:

```sh
lumo version
lumo new
```

**Node ≥ 18** is required. Supported platforms are exactly the release
matrix (`os`: linux/darwin/win32, `cpu`: x64/arm64) — anything else
(e.g. Windows on ARM) fails npm's standard `EBADPLATFORM` check at
install time by design, because no release archive exists for it.

## How it works

The package itself is ~5 kB. It contains no binary:

1. **`postinstall`** (`scripts/postinstall.js`) downloads
   `lumo_v<version>_<os>_<arch>.{tar.gz,zip}` and `SHA256SUMS.txt` from
   the matching [GitHub release](https://github.com/intruder0007/Lumo/releases),
   verifies the archive's SHA-256 against the published checksum file,
   and extracts just the `lumo`/`lumo.exe` binary into
   `distribution/npm/.bin/` (inside the installed package). The release
   binary embeds the V1 plugin set (ADR-0012), so the installed CLI is
   self-sufficient.
2. **`bin`** (`bin/lumo.js`) spawns that binary with
   `stdio: 'inherit'` and forwards argv and the exit code exactly —
   required for the interactive wizard's raw-mode arrow-key UI.
3. **Idempotency marker.** A `.lumo-version` marker next to the
   binary records which package version was installed; re-installing
   the same version (npm ci, cache reuses) skips the download. The
   marker is only written after a successful install, and the binary
   itself is checked to still exist before a skip — a deleted or
   corrupted binary always re-downloads.

The binary is **not** bundled into the tarball: this keeps the package
tiny and avoids shipping five platform binaries to every user. The
trade-off is that `npm install` needs network access to GitHub and runs
a lifecycle script. If install scripts were skipped (`--ignore-scripts`
/ `--no-optional`), the `lumo` command fails with an actionable
message telling you exactly how to fix it.

## Versioning and release coupling

The package version **always equals the `lumo` release version it
launches** (ADR-0015). `lumo-cli@1.0.0` installs `lumo`
v1.0.0, full stop. Three guards enforce this:

- `release.yml`'s `publish-npm` job fails unless
  `distribution/npm/package.json`'s version equals the git tag.
- `npm publish` runs `prepublishOnly` → `scripts/verify-release.js`,
  which checks that every platform archive **and** `SHA256SUMS.txt`
  exists for this version before anything is published.
- The postinstall marker and download target both derive from the
  package version, so a mismatch would fail loudly at install.

Do not bump the wrapper version to anything other than the next
`lumo` release version.

## Publishing (maintainers)

Publishing is automatic: cutting a `vX.Y.Z` tag runs
`.github/workflows/release.yml`, which creates the GitHub release and —
**after** it, so the assets the installer needs exist — publishes this
package to npm with provenance signing. Requirements:

- The `NPM_TOKEN` repository secret (an npm *automation* token with
  publish rights for `lumo-cli`). Without it the publish job
  is skipped, visibly, and the GitHub release still succeeds.
- `distribution/npm/package.json`'s version bumped to the tag version
  in the same commit that updates `CHANGELOG.md`.

Full runbook, rollback, and deprecation instructions:
[`docs/guides/releasing.md`](../../docs/guides/releasing.md).

## Status

**Live on npm as `lumo-cli@0.4.0`** (published 2026-08-03 by
`release.yml` on the v0.4.0 tag, version-synced). The interim package
`bootstrap-cli-dev@0.3.0` (published under the platform's old name,
2026-08-02) remains on npm, deprecated 2026-08-02 with a pointer to
this package; see ADR-0015 and the migration audit in
`docs/architecture/npm-identity-migration.md`. Publishing is automatic via
`.github/workflows/release.yml` (requires the repo's `NPM_TOKEN`
secret, which is configured). The registry-side check
(`.github/workflows/npm-verify-published.yml`) verifies the published
package on clean Linux and Windows runners on demand.
