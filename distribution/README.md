# Distribution wrappers

Per-ecosystem thin launchers so users can install `bootstrap` through
their existing tooling. See
[ADR-0010](../docs/architecture/adr/0010-distribution-architecture.md)
and the [wrapper protocol](../docs/architecture/distribution-protocol.md)
— every wrapper follows the same contract (resolve platform →
locate/fetch the release archive as a whole → exec `bootstrap` with
stdio/argv/exit-code passed through unmodified).

7 of 8 are now built and verified (locally and/or in
`.github/workflows/distribution-verify.yml`, one job per ecosystem, each
against real published `v0.3.0` release assets) — **none are published**
to any real registry. Publishing needs either a real credential (npm,
PyPI, Cargo, Chocolatey) or a PR to a third-party-owned repo
(Homebrew, Scoop, Winget) — both are separate, explicitly-confirmed
future steps per ecosystem, not bundled into building/verifying.

| Ecosystem | Directory | Status |
|---|---|---|
| npm | [`npm/`](npm/) | built, verified locally + CI |
| PyPI | [`pypi/`](pypi/) | built, CI-verified |
| Cargo | [`cargo/`](cargo/) | built, CI-verified (lowest priority — see its README) |
| Go | [`go/`](go/) | solved (ADR-0012) |
| Homebrew | [`homebrew/`](homebrew/) | built, CI-verified |
| Scoop | [`scoop/`](scoop/) | built, CI-verified |
| Winget | [`winget/`](winget/) | built, verified locally + CI |
| Chocolatey | [`chocolatey/`](chocolatey/) | built, CI-verified |
