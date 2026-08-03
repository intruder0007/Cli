# Security policy

Lumo takes security seriously. This document explains what "secure" means
for Lumo, what guarantees we provide, what we don't, how to report a
vulnerability, and which versions receive fixes.

## Supported versions

Only the latest minor release line receives security fixes. A new minor
bump supersedes the previous; patch fixes backport only when a fix is both
low-risk and not already superseded.

| Version | Supported |
|---|---|
| `v0.3.x` | ✅ (latest release line at time of writing) |
| `< v0.3.0` | ❌ — upgrade to the latest release |
| `main` (unreleased) | Best-effort; not a release channel |

When `v1.0.0` ships, this table updates to commit to a maintenance window
for the previous minor (typically one minor back, security-only).

## Reporting a vulnerability

**Please do not open a public GitHub issue for a security vulnerability.**

- Email the maintainer (see the GitHub profile for the address on
  `github.com/intruder0007`), or
- Use GitHub's **Security Advisories** (Report a vulnerability) on the
  repository (`Security` → `Advisories` → `New draft security advisory`),
  which keeps the report private to repository admins.

Include: affected version, the surface (CLI / wire protocol / a specific
plugin / a distribution wrapper / the npm package), reproduction steps, and
your assessment of impact. You will get an acknowledgement within **72
hours** and a fix-or-explanation update within **14 days**. A public CVE is
requested by the reporter and maintainer jointly; coordinated disclosure
follows. Reporters are credited in the advisory unless they prefer
otherwise.

## Threat model at a glance

Lumo is a **scaffolding orchestrator**: it runs plugin binaries as
subprocesses and writes files into a directory the user points it at. The
trust boundaries are:

### 1. Plugin execution — you run code you installed

A template or capability plugin is a **compiled binary** that Lumo spawns
as a subprocess. **Installing a plugin is consent to run that code** with
the full privileges of your user account (filesystem, network,
environment). Lumo does **not** sandbox plugins — doing so meaningfully
across Linux/macOS/Windows is out of scope for v1.

What we *do* guarantee:

- **Manifest validation.** Every plugin ships a `plugin.json` that
  `core/registry` validates; a malformed manifest is rejected up front,
  never executed.
- **Protocol handshake.** The host and plugin must agree on
  `protocolVersion` at `plugin.initialize`; a mismatched binary is
  rejected before any `generate`/`apply` runs (`plugin.ProtocolMismatchError`).
- **Fail-fast.** The built-in `lumo plugins validate <dir>` proves a
  plugin's entrypoint spawns and passes the handshake *before* you ship it.
- **No auto-execution of arbitrary code.** Plugins are discovered by
  directory scan only — Lumo never downloads and runs a binary from the
  network on its own. A plugin on disk was put there by you (or by a
  package manager you chose).

What you must accept: treat plugin installation the way you treat
`npm install` of a postinstall script — only install plugins from authors
you trust. The remote plugin registry/marketplace (a future effort) will add
its own trust model; today the boundary is local-discovery-only by design.

### 2. Distribution transport integrity

Every release channel verifies integrity:

- The **npm wrapper** (`lumo-cli`/`bootstrap-cli-dev`) downloads the
  release archive over HTTPS, verifies the archive's SHA-256 against the
  published `SHA256SUMS.txt`, and extracts only the `lumo`/`lumo.exe`
  binary. A checksum mismatch aborts the install loudly; it never produces
  a partial or substituted binary.
- The **script installers** (`install.sh`, `install.ps1`) fetch over HTTPS
  and (planned for v0.5.0) verify against `SHA256SUMS.txt`.
- Release **archives** are produced by `.github/workflows/release.yml`
  (cross-compiled on GitHub-hosted runners) and the resulting
  `SHA256SUMS.txt` is published alongside them on the GitHub Release.

### 3. Provenance & signing

- The npm package is published with **npm provenance** (Sigstore-based,
  via GitHub OIDC) where the publish happens from CI — `npm install`
  users can verify the provenance bundle with `npm view
  lumo-cli@<version> --json` (the `_provenance`/`distro` field).
- Release-archive-level detached signatures (cosign/Sigstore over the
  `SHA256SUMS.txt`) are a v0.8.0 (RC) goal; until then, the GitHub Release
  assets' integrity rests on GitHub's TLS + the published checksums, and
  npm rests on provenance + checksum verification.

### 4. Secrets

- The `NPM_TOKEN` repository secret (an npm *automation* token) is used by
  `release.yml`'s `publish-npm` job only; it is never printed in logs,
  never embedded in the published package, and lives only in the repo's
  Actions secrets.
- Lumo itself stores no secrets, makes no network calls during project
  generation, and the `--verbose`/`LUMO_DEBUG` diagnostic output is
  audited to avoid leaking user-private paths or token material.

## What is out of scope

- Sandboxed plugin execution (see §1). A future ADR may add an opt-in
  restricted profile, but v1 plugins are trusted code.
- A remote plugin registry/marketplace trust model — deferred to a v2
  effort with its own ADR.
- Self-update of the installed binary — the CLI is static; updates come
  through your package manager / re-install (auto-update is a non-goal,
  see `docs/architecture/roadmap.md`).

## Disclosure policy / safe harbor

We will not pursue legal action against good-faith security research that
respects the reporting channel and avoids user-data exposure or service
disruption. We ask the same in return: coordinated disclosure, no public
drop before a fix or an agreed timeline, and a CVE jointly requested.
