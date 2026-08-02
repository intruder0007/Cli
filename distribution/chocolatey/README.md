# Chocolatey package — built, CI-verified, not published

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).

`lumo.nuspec` + `tools/chocolateyinstall.ps1` using
`Install-ChocolateyZipPackage` against the real `v0.3.0`
`lumo_v0.3.0_windows_amd64.zip` release asset, with its real checksum
(from the published `SHA256SUMS.txt`) passed as `-Checksum`, extracting
to `$toolsDir` — the whole archive, not just the binary, so
`templates/`/`plugins/builtin` land alongside it — and letting
Chocolatey's shim generator pick up `lumo.exe` from there
automatically.

**Verified in CI** (`.github/workflows/distribution-verify.yml`,
`chocolatey` job, `windows-latest` — Chocolatey ships preinstalled on
this runner image): `choco pack` followed by `choco install
lumo-cli --source distribution\chocolatey -y` and a real
`lumo version` invocation. Not verified locally — Chocolatey isn't
installed in this repo's own dev environment.

**Not published.** `choco push` needs a real Chocolatey.org API key,
which isn't available in the environment that built this.
