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
against real published release assets) — **one is live on npm: the
interim package** `bootstrap-cli-dev@0.3.0`, published 2026-08-02
under the platform's old name (ADR-0015), deprecated 2026-08-02.
The renamed wrapper
(`lumo-cli`, version-synced with the `lumo` release it launches) is
published as `lumo-cli@1.0.0` by `.github/workflows/release.yml` when
the v1.0.0 tag is cut, replacing the interim package
(see `npm/README.md`). The
remaining ecosystems are built and CI-verified but not published:
publishing needs either a real credential (PyPI, Cargo, Chocolatey) or
a PR to a third-party-owned repo (Homebrew, Scoop, Winget) — separate,
explicitly-confirmed future steps per ecosystem.

| Ecosystem | Directory | Status |
|---|---|---|
| npm | [`npm/`](npm/) | `bootstrap-cli-dev@0.3.0` live, deprecated (old name, 2026-08-02); `lumo-cli@1.0.0` publish pending the v1.0.0 release |
| PyPI | [`pypi/`](pypi/) | built, CI-verified, not published |
| Cargo | [`cargo/`](cargo/) | built, CI-verified, not published (lowest priority — see its README) |
| Go | [`go/`](go/) | solved (ADR-0012) |
| Homebrew | [`homebrew/`](homebrew/) | built, CI-verified, not published |
| Scoop | [`scoop/`](scoop/) | built, CI-verified, not published |
| Winget | [`winget/`](winget/) | built, verified locally + CI, not published |
| Chocolatey | [`chocolatey/`](chocolatey/) | built, CI-verified, not published |

Cutting a release (GitHub release + npm publish + verification):
[`docs/guides/releasing.md`](../docs/guides/releasing.md).
