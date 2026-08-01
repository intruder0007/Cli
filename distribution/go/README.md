# Go (`go install`) — solved

Not a wrapper: `cli` is already a public Go module, so
`go install github.com/intruder0007/Cli/cli@latest` compiles and
installs a real, working `bootstrap` binary. No new code needed for
that part.

**The former gap is closed** (ADR-0012): `go install` still produces
only the binary — no sibling `templates/`/`plugins/` directories — but
the binary now embeds the V1 plugin set at build time and
self-extracts it to a cache directory on first use whenever no sibling
directories are found. See the
[wrapper protocol](../../docs/architecture/distribution-protocol.md)'s
`go install` section and
[ADR-0012](../../docs/architecture/adr/0012-universal-install-architecture.md).
No `CLI_PLUGIN_DIRS` workaround is needed anymore.
