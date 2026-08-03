# Distribution wrappers

Per-ecosystem thin launchers so users can install `lumo` through
their existing tooling. See
[ADR-0010](../docs/architecture/adr/0010-distribution-architecture.md)
and the [wrapper protocol](../docs/architecture/distribution-protocol.md)
— every wrapper follows the same contract (resolve platform →
locate/fetch the release archive as a whole → exec `lumo` with
stdio/argv/exit-code passed through unmodified).

7 of 8 are built and verified (locally and/or in
`.github/workflows/distribution-verify.yml`, one job per ecosystem, each
against real published release assets) — **npm is live with the renamed
package**: `lumo-cli@0.4.0`, published 2026-08-03 by
`.github/workflows/release.yml` on the v0.4.0 tag, version-synced with
the release it launches. The interim `bootstrap-cli-dev@0.3.0` package
(old name, ADR-0015) is deprecated on npm since 2026-08-02 and points
users at `lumo-cli`. The
remaining ecosystems are built and CI-verified but not published:
publishing needs either a real credential (PyPI, Cargo, Chocolatey) or
a PR to a third-party-owned repo (Homebrew, Scoop, Winget) — separate,
explicitly-confirmed future steps per ecosystem.

| Ecosystem | Directory | Status |
|---|---|---|
| npm | [`npm/`](npm/) | `lumo-cli@0.4.0` live (published 2026-08-03, version-synced); interim `bootstrap-cli-dev@0.3.0` deprecated 2026-08-02 |
| PyPI | [`pypi/`](pypi/) | built, CI clean-install verified against the v0.4.0 release assets; not published |
| Cargo | [`cargo/`](cargo/) | built, CI clean-install verified against the v0.4.0 release assets (lowest priority — see its README) |
| Go | [`go/`](go/) | solved (ADR-0012) |
| Homebrew | [`homebrew/`](homebrew/) | built, CI clean-install verified against the v0.4.0 release assets; not published |
| Scoop | [`scoop/`](scoop/) | built, CI clean-install verified against the v0.4.0 release assets; not published |
| Winget | [`winget/`](winget/) | built, CI clean-install verified against the v0.4.0 release assets; not published |
| Chocolatey | [`chocolatey/`](chocolatey/) | built, CI clean-install verified against the v0.4.0 release assets; not published |

> **Re-verification state.** Every wrapper is pinned at the latest
> published release (`v0.4.0`, `lumo_v0.4.0_*` assets) with real
> checksums from the release's `SHA256SUMS.txt`. The non-npm wrappers
> are repinned (URL + checksum) whenever a release is cut; the npm
> wrapper re-verifies its pinned version against the release assets at
> publish time (`scripts/verify-release.js`) and in
> `distribution-verify.yml`. `distribution-verify.yml`'s clean-install
> jobs run against these real assets — green means the manifest was
> clean-installed and a real `lumo version` ran on a disposable runner.

Cutting a release (GitHub release + npm publish + verification):
[`docs/guides/releasing.md`](../docs/guides/releasing.md).
