# ADR-0016: npm identity migration — `bootstrap-cli-dev` → `lumo-cli`

## Status

Accepted

## Context

ADR-0015 established the first live distribution channel by publishing
the npm wrapper as `bootstrap-cli-dev@0.3.0` on 2026-08-02 — under the
platform's old name, because the rename to `Lumo` happened after that
publish. The wrapper itself was already renamed in the tree
(`distribution/npm/package.json` is `lumo-cli`, version `1.0.0`, with
the full production metadata ADR-0015 requires), but it has never been
published under that name: `lumo-cli` is unclaimed on npm (verified at
audit time) and the registry still only knows `bootstrap-cli-dev@0.3.0`.

Publishing under the right name also binds the version-sync rule:
ADR-0015 says the wrapper version *always equals* the release version
it launches, and the tree's wrapper is at `1.0.0` while the latest
release is `v0.3.0`. So the migration publish cannot happen before the
v1.0.0 GitHub release exists — the pre-publish guard
(`scripts/verify-release.js`, run by `prepublishOnly` and CI) enforces
this by failing `npm publish` until every `lumo_v1.0.0_*` archive and
`SHA256SUMS.txt` exist.

A second, separate discovery during the audit: the distribution
metadata files carried corruption from an earlier bulk-edit —
single-letter substitutions in prose and metadata ("aauncher",
"aumo", "hhe", "hhin", "EBADPLAhFORM", `license = "MIh"`, the
`NPM_hOKEN` secret name in a README) plus stale "published" claims
that described `lumo-cli@0.3.0` as live. Left in place, a publish
would have shipped wrong repository/bugs/homepage metadata and
misleading registry prose.

## Decision

**Publish the npm wrapper as `lumo-cli@1.0.0` at v1.0.0 release time,
replacing the interim `bootstrap-cli-dev@0.3.0`, which is deprecated
now (2026-08-02) rather than at publish time.**

- **Name `lumo-cli`.** Unclaimed on npm (checked 2026-08-02); matches
  the product identity the platform now carries; unscoped, public
  `publishConfig.access`. The old name stays deprecated on the registry
  as a pointer so nobody re-creates it as a confusing near-identical
  package — deprecating now, before the successor exists, is safe
  because the deprecation message tells users the interim package
  keeps working until the successor publishes.
- **Publish gate unchanged: GitHub release first, npm second.** The
  existing `release.yml` `publish-npm` job (version-sync check,
  `prepublishOnly` guard, provenance) publishes `lumo-cli@1.0.0` when
  the v1.0.0 tag is cut. No manual registry step is needed or allowed.
- **No other ecosystem is affected.** PyPI/Cargo/Homebrew/Scoop/Winget/
  Chocolatey were never published under the old name; their manifests
  already use `lumo-cli`/`Lumo` identifiers (their descriptions are
  fixed to the canonical "Lumo project scaffolding platform" prose as
  part of this ADR's metadata repair).
- **Corrupted and stale metadata is corrected in this migration, not
  shipped.** The audit's findings (below) are fixed in the tree; the
  fixed `package.json` is what the registry will show.

## Consequences

- `bootstrap-cli-dev@0.3.0` is deprecated with the message:
  "Renamed to lumo-cli: the successor package publishes as
  lumo-cli@1.0.0 with the platform v1.0.0 release. Until that release
  this interim package continues to work; afterwards install lumo-cli
  instead." Deprecation is reversible (`npm deprecate` with an empty
  message) if the publish plan changes.
- Until v1.0.0 exists, `npm view lumo-cli` fails and
  `.github/workflows/npm-verify-published.yml` must not be run (it
  installs `lumo-cli@latest`; it is only meaningful after the first
  publish). The `distribution-verify.yml` wrapper jobs already point at
  the v1.0.0 assets and remain blocked on the same release.
- Users of `bootstrap-cli-dev@0.3.0` see the deprecation warning at
  install time with a correct pointer; the package itself still works
  (postinstall downloads `v0.3.0` assets that exist).
- The registry metadata for `lumo-cli@1.0.0` will be the corrected
  `package.json`: description ("Launcher for the Lumo project
  scaffolding platform…"), homepage/bugs/repository →
  `github.com/intruder0007/Lumo`, license `MIT` — all verified by the
  audit's final corruption sweep (zero matches) and by `npm pack`
  contents inspection.
- Verified before acceptance: `npm pack` produces exactly the five
  intended files; `verify-release.js` fails correctly with the six
  404s for the missing v1.0.0 assets (exit 1, actionable message);
  a sandbox install of the packed tarball fails loudly and cleanly at
  postinstall (HTTP 404) instead of producing a broken shim; with
  `--ignore-scripts` the shim reports the actionable missing-binary
  error. Full audit record: `docs/architecture/npm-identity-migration.md`.
