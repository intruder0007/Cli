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
  design, but only `sdk/go` ships in V1. A Python/Node/Rust SDK is
  additive, not a breaking change.
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
- **Workspace/monorepo project generation.** V1 generates exactly one
  project per run.
- **Localization/i18n of the wizard.** English only in V1.
- **Distribution wrapper implementations.** ADR-0010 designed the
  contract and scaffolded `distribution/<ecosystem>/`; no npm, PyPI,
  Cargo, Homebrew, Scoop, Winget, or Chocolatey wrapper is actually
  built yet. `go install` already works but is missing plugins (see
  `distribution/go/README.md`).
