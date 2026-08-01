# Go (`go install`) — mostly solved, one documented gap

Not a wrapper: `cli` is already a public Go module, so
`go install github.com/intruder0007/Cli/cli@latest` already compiles and
installs a real, working `bootstrap` binary today. No new code needed
for that part.

**The gap**: `go install` produces only the binary — no sibling
`templates/`/`plugins/` directories, which `pluginDirs()` needs (see the
[wrapper protocol](../../docs/architecture/distribution-protocol.md)'s
"Known gap" section and [ADR-0010](../../docs/architecture/adr/0010-distribution-architecture.md)).
Until that's resolved (`go:embed` self-extraction is the recommended
direction), `go install` users need `CLI_PLUGIN_DIRS` pointed at a
manually-downloaded release archive, or should install a release archive
directly instead.
