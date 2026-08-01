# Winget manifest (not implemented)

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).

A Winget manifest (`InstallerType: zip`, pointing at
`cli_<version>_windows_amd64.zip`, `InstallerSha256` from
`SHA256SUMS.txt`, `NestedInstallerFiles: [{ RelativeFilePath:
bootstrap.exe, PortableCommandAlias: bootstrap }]`). Winget's zip
installer type extracts the whole archive to the package's install
directory and only aliases the named file onto PATH — same
whole-archive-survives property as Scoop, no custom logic needed.
