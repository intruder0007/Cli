# Homebrew formula — built, CI-verified, not published

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).
Simplest of the 8 — no wrapper process needed at all.

`lumo.rb`: per-`on_macos`/`on_linux` + per-`Hardware::CPU.arm?`
`url`/`sha256` blocks pointing at the real `v0.3.0` release assets and
their real checksums (from the published `SHA256SUMS.txt`). `install`
stages the **whole** extracted directory into `libexec` and symlinks
only `lumo` into `bin` — the simplified `bin.install "lumo"`
that ADR-0012's embedded-plugin fallback eventually allows isn't used
because `v0.3.0` ships the sibling `templates/`/`plugins/builtin/`
directories anyway and the whole-archive pattern stays the conservative
choice (see the formula's own comment, and ADR-0012's "don't simplify
the archive without a measurable need" decision). Includes a `test do`
block that runs `lumo version` and `lumo doctor`.

**Verified in CI** (`.github/workflows/distribution-verify.yml`,
`homebrew` job, `macos-latest`): `brew install --formula
distribution/homebrew/lumo.rb` followed by `brew test lumo`
and a real `lumo version` invocation. Not verified locally — this
repo's dev environment is Windows, no Homebrew there.

**Not published.** Submitting to `homebrew-core` (strict acceptance
criteria) or creating a new public tap under the maintainer's account
both mean opening a PR against a real, third-party or newly-public repo
— a visible, hard-to-reverse action requiring explicit go-ahead in a
separate step, not done here.
