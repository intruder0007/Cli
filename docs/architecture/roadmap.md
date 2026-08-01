# Roadmap: explicitly deferred beyond V1

Everything below is a deliberate non-goal for V1. It's listed here (rather
than left implicit) so the boundary between "not built yet" and "not
planned" is clear, per [ADR-0004](adr/0004-v1-scope.md).

- **Remote plugin/template registry & marketplace.** V1 discovery is a
  local directory scan only (`docs/architecture/plugin-protocol.md`
  "Discovery"). `core/registry`'s interface is written so a remote source
  can be added as another implementation later without changing callers.
- **Additional project types, languages, and frameworks.** The wizard
  already lists them as "(coming soon)"; adding one is "write a new
  template plugin," not a core change (ADR-0002).
- **Non-Go plugin transports.** WASM and native dynamic library transports
  were considered and rejected for V1 (ADR-0002); subprocess+JSON-RPC
  stays the only transport unless a concrete need forces revisiting that
  ADR.
- **Non-Go plugin SDKs.** The wire protocol is language-agnostic by
  design, and the SDK family's language-neutral spec now exists
  (`docs/architecture/sdk-architecture.md`, ADR-0014) with design
  notes under `sdk/{node,python,rust,future}/` — but only `sdk/go`
  ships. Implementing one is additive, not a breaking change; each
  needs an implementation phase with CI-verified clean-machine
  round-trip (the same bar `distribution-verify.yml` applies to
  wrappers).
- **Richer CLI theming.** V1 ships exactly two themes (`default`,
  `minimal`); a theme is just a set of rendering choices in
  `cli/internal/prompt`, so more can be added without touching `core`.
- **Telemetry/analytics.** Offline-first (project principle) means no
  telemetry is collected by default in V1 or planned to be added silently
  later; if this ever changes it needs its own ADR and explicit opt-in.
- **Auto-update.** The CLI is a static binary users fetch themselves; no
  self-update mechanism in V1.
- **Capability conflict detection.** ADR-0008 added `dependsOn`-based
  ordering, but there's still no detection of two capabilities that
  touch the same file in incompatible ways — ordering, not conflict
  resolution.
- **Capability-to-capability visibility.** A capability plugin's
  `plugin.apply` request doesn't include which *other* capabilities were
  selected — only `dependsOn` ordering exists, not awareness. No V1
  capability needs this yet; would need a `plugin-protocol.md` change
  (and likely its own ADR) if a concrete need arises.
- **Workspace/monorepo project generation.** V1 generates exactly one
  project per run.
- **Localization/i18n of the wizard.** English only in V1.
- **Distribution wrapper *publishing*.** The 7 wrappers (npm, PyPI,
  Cargo, Homebrew, Scoop, Winget, Chocolatey) are built and CI-verified
  against real release assets, but none is published to a live registry
  — publishing needs either a credential (npm/PyPI/Cargo/Chocolatey) or
  a PR to a third-party repo (Homebrew/Scoop/Winget), each an
  explicitly-confirmed separate step. `go install` works correctly out
  of the box as of ADR-0012's embedded plugin fallback (see
  `distribution/go/README.md`).
- **Multi-version protocol negotiation.** The host supports exactly one
  wire-protocol version per release and rejects mismatches at the
  `plugin.initialize` handshake (ADR-0008); serving two protocol
  versions simultaneously (so old plugins keep working across a
  protocol bump) is deferred until a protocol change actually needs it
  — see `api-compatibility.md`.
- **Public theme API.** The theme registry (`RegisterTheme`/`GetTheme`
  in `cli/internal/prompt`) is a real in-process seam (ADR-0007) but
  lives in an `internal/` package, so it's Experimental, not public.
  Promoting it to a Stable public API — or adding theme *plugins* — is
  a future ADR, needed only when a concrete third-party theme exists
  (see ADR-0013).
- **Old-version embedded-cache cleanup.** ADR-0012's self-extracted
  plugin fallback is scoped per version
  (`os.UserCacheDir()/bootstrap/<version>/`), so upgrading never serves
  stale plugins, but nothing garbage-collects a *previous* version's
  cache directory. Not proven to matter yet.
- **Release archive simplification.** Now that the `cli` binary is
  self-sufficient (ADR-0012), the release archive's sibling
  `templates/`/`plugins/builtin/` directories are technically
  redundant. Left as-is deliberately — every existing distribution doc
  and would-be package-manager formula assumes today's archive shape;
  revisit only once the embedded fallback has real-world mileage.
