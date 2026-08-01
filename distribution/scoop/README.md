# Scoop manifest — built, CI-verified, not published

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).

`bootstrap.json`: `url`/`hash` pointing at the real `v0.3.0`
`cli_v0.3.0_windows_amd64.zip` release asset and its real checksum (from
the published `SHA256SUMS.txt`), `bin: "bootstrap.exe"`. Scoop extracts
the whole zip into the app's versioned directory by default and only
*shims* the named `bin` — it doesn't discard the rest, so
`templates/`/`plugins/builtin` stay alongside `bootstrap.exe`
automatically. No custom install script needed.

**Verified in CI** (`.github/workflows/distribution-verify.yml`,
`scoop` job, `windows-latest`): installs Scoop fresh on the runner,
then `scoop install .\distribution\scoop\bootstrap.json` followed by a
real `bootstrap version` invocation. Not verified locally — Scoop isn't
installed in this repo's own dev environment, and installing it there
would modify the actual dev machine rather than a disposable runner.

**Not published.** Submitting to the community `extras` bucket
(`ScoopInstaller/Extras`) or creating a personal bucket (a new public
repo) both mean a visible, hard-to-reverse third-party-facing action
requiring explicit go-ahead in a separate step, not done here.
