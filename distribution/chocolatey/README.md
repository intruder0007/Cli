# Chocolatey package (not implemented)

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).

A Chocolatey `.nuspec` + `chocolateyinstall.ps1` using
`Install-ChocolateyZipPackage` against `cli_<version>_windows_amd64.zip`
(with its `SHA256SUMS.txt` checksum passed as `-Checksum`), extracting
to `$toolsDir` and letting Chocolatey's shim generator pick up
`bootstrap.exe` from there — again, extract the whole archive, not just
the binary, so `templates/`/`plugins/` land alongside it.
