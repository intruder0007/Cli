# Scoop manifest (not implemented)

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).

A Scoop JSON manifest (`url`, `hash` pointing at
`cli_<version>_windows_amd64.zip` and its `SHA256SUMS.txt` entry, `bin:
"bootstrap.exe"`). Scoop extracts the whole zip into the app's versioned
directory by default and only *shims* the named `bin` — it does not
discard the rest of the archive, so `templates/`/`plugins/` stay
alongside `bootstrap.exe` automatically. No custom install script
needed.
