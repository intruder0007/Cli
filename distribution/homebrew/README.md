# Homebrew formula (not implemented)

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).
Simplest of the eight — no wrapper process needed at all.

A standard Homebrew formula with per-`on_macos`/`on_linux` +
per-`Hardware::CPU.arm?` `url`/`sha256` blocks pointing at the existing
`cli_<version>_<os>_<arch>.tar.gz` release assets and their
`SHA256SUMS.txt` checksums (Homebrew fetches+verifies natively — no
custom download code needed), with `install` staging the **whole**
extracted directory into `libexec` and symlinking only `bootstrap` into
`bin` (a bare `bin.install "bootstrap"` would strand the sibling
`templates/`/`plugins/` — see the protocol doc's "Why the whole
archive").
