# Distribution wrappers

Per-ecosystem thin launchers so users can install `bootstrap` through
their existing tooling. See
[ADR-0010](../docs/architecture/adr/0010-distribution-architecture.md)
and the [wrapper protocol](../docs/architecture/distribution-protocol.md)
before implementing any of these — every wrapper must follow the same
contract (resolve platform → locate/fetch the release archive as a
whole → exec `bootstrap` with stdio/argv/exit-code passed through
unmodified). None of these are implemented yet; each subdirectory is a
design note for whoever picks it up.

| Ecosystem | Directory | Status |
|---|---|---|
| npm | [`npm/`](npm/) | not started |
| PyPI | [`pypi/`](pypi/) | not started |
| Cargo | [`cargo/`](cargo/) | not started |
| Go | [`go/`](go/) | mostly solved already; documented gap remains |
| Homebrew | [`homebrew/`](homebrew/) | not started |
| Scoop | [`scoop/`](scoop/) | not started |
| Winget | [`winget/`](winget/) | not started |
| Chocolatey | [`chocolatey/`](chocolatey/) | not started |
