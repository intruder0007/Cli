# Winget manifest — built, locally validated, CI-verified, not published

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).

3-file manifest set under
`manifests/i/intruder0007/Lumo/0.3.0/` (version, installer, locale —
`InstallerType: zip`, `InstallerSha256` from the real published
`SHA256SUMS.txt`, `NestedInstallerFiles: [{ RelativeFilePath:
lumo.exe, PortableCommandAlias: lumo }]`). Winget's zip
installer type extracts the whole archive to the package's install
directory and only aliases the named file onto `PATH` — same
whole-archive-survives property as Scoop, no custom logic needed.

**Verified locally**: `winget validate
distribution\winget\manifests\i\intruder0007\Lumo\0.3.0` passed against
the real `winget` on this dev machine (schema/lint check only, no
install).

**Verified in CI** (`.github/workflows/distribution-verify.yml`,
`winget` job, `windows-latest`): `winget validate` + `winget install
--manifest` + a real `lumo version` invocation — skipped
gracefully if `winget` isn't fully available/configured on that runner
image, since this isn't guaranteed across all GitHub-hosted Windows
runners.

**Not published.** Submission is a PR to `microsoft/winget-pkgs` (their
own validation pipeline runs on the PR) — a visible, hard-to-reverse
third-party-facing action requiring explicit go-ahead in a separate
step, not done here.
