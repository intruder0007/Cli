# Migration guide

Every user-visible breaking change, and how to move between releases.
New entries follow the same format (per
[`api-compatibility.md`](../architecture/api-compatibility.md), step 4):
**version — what changed — who is affected — what to do.**

The policy behind this: breaking changes to a Stable surface arrive
only with a Major (or, at `v0.x`, a Minor) release, are announced in
`CHANGELOG.md`, and are deprecated for at least one full minor release
first. Anything not listed here is additive and needs no action.

## History

### v0.1.1 — no user-visible breaking changes

Patch release; bug fixes only.

### v0.2.0 — plugin discovery and handshake hardening (ADR-0008)

**What changed:** plugin discovery and the handshake were hardened.

- A plugin directory whose `plugin.json` fails to parse or fails
  `Manifest.Validate()` is now **skipped with a reported issue**
  instead of failing discovery of everything else. `bootstrap plugins
  list` prints what was skipped and why, to stderr.
- The host now **cross-checks identity and protocol version** at the
  `plugin.initialize` handshake: a plugin process whose self-reported
  `name` or `protocolVersion` disagrees with the manifest on disk is
  rejected (`plugin.ProtocolMismatchError` /
  `plugin.IdentityMismatchError`), and the template/capability is
  never invoked.

**Who is affected:** anyone relying on either of the behaviors above.

- Anyone whose plugin had an invalid manifest (it was previously a
  hard failure; now it is silently skipped — check `bootstrap plugins
  list` stderr output).
- Anyone with a stale or swapped binary sitting at a plugin's
  entrypoint path (previously misbehaved at generate/apply time;
  now it is rejected up front).

**What to do:**

- Fix invalid manifests (`sdk.Manifest.Validate()` is the rule set).
- Run `bootstrap plugins validate <dir>` on any plugin you maintain
  (see the authoring guides) — it performs exactly this check.

### v0.3.0 — wrapper repoint

**What changed:** the wrappers were repointed to the new release.

- The `distribution/` wrappers (homebrew, scoop, winget, chocolatey,
  cargo, npm, pypi) were repointed from `v0.2.0` to `v0.3.0`.

**Who is affected:** wrapper users updating to the latest release.

**What to do:** nothing — the wrapper contract is unchanged; this is a
version bump that includes the accumulated fixes.

### v0.4.0+ (unreleased) — ecosystem phases

Everything shipped since `v0.3.0` — the API-compatibility policy
(ADR-0013), the SDK foundation (ADR-0014), `bootstrap plugins
validate`, and the wire-protocol golden tests — is **additive**:
nothing listed here requires an action from existing users, plugin
authors, or wrapper maintainers.

## Pre-flight checklist (any upgrade)

1. `bootstrap version` — confirm the installed version.
2. `bootstrap doctor` — local health checks (plugin directory setup,
   wrapper setup).
3. `bootstrap plugins list` — confirm every plugin you rely on
   discovers cleanly (read the stderr for skipped plugins).
4. If you maintain plugins: `bootstrap plugins validate <dir>` for each
   of them.

## Worked example: a future wire-protocol change

This is the shape every future entry takes. Suppose `protocolVersion`
moves `"1"` → `"2"` with a changed `plugin.generate` response:

> **v1.1.0 — wire protocol v2 (breaking)**
>
> **What changed**: `plugin.initialize` now carries a `versions`
> array; `plugin.generate`'s response renames `filesWritten` to
> `created`. `sdk.ProtocolVersion` is `"2"`; the v1 manifest
> (`protocolVersion: "1"`) is deprecated but still served for one
> full minor release.
>
> **Who is affected**: plugins built against `sdk/go` v0.x that
> declare `protocolVersion: "1"`.
>
> **What to do**: update the SDK, bump `protocolVersion` in
> `plugin.json`, re-run `bootstrap plugins validate <dir>`, and
> update any code reading the renamed field.
