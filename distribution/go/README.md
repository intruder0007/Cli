# Go (`go install`) — binary yes, embedded fallback not yet

`go install github.com/intruder0007/Lumo/cli@latest` compiles and
installs the `lumo` binary itself. Two things keep it from being a
drop-in "works out of the box" install today:

1. **Tags.** The `cli` directory is its own Go module
   (`cli/go.mod`), so `@latest` (or `@cli/vX.Y.Z`) resolves only once
   `cli/vX.Y.Z` submodule tags are published. None exist as of v0.3.0 —
   `go install ...@latest` fails with "no matching versions".
2. **Embedded plugin fallback.** ADR-0012's fallback works by *staging*
   built plugin binaries into `cli/internal/embedded/assets/` — a
   gitignored directory (`.gitignore`, "Plugin binaries + manifests
   staged for go:embed") — immediately before `go build`. Only the
   `Makefile`'s `stage-embedded`/`build` targets and the release
   pipeline (`.github/workflows/release.yml`) do that staging; `go
   install` has no hook for it. A `go install`-produced binary embeds
   only the `.gitkeep` placeholder, so `lumo doctor` reports "no
   plugin assets embedded", and `lumo new` works only when sibling
   `templates/`/`plugins/builtin/` directories (or `LUMO_PLUGIN_DIRS`)
   are present — the pre-ADR-0012 situation.

**What does work out of the box:** every binary the release pipeline
builds — the release archives, and therefore `install.sh`/`install.ps1`
and all 7 distribution wrappers — carries the embedded plugin set
(ADR-0012's "Build-order coupling"). Those are the supported install
paths until the two gaps above are closed; see the wrapper protocol's
`go install` section and
[ADR-0012](../../docs/architecture/adr/0012-universal-install-architecture.md).
