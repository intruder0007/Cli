$ErrorActionPreference = 'Stop'

# Extracts the whole archive into $toolsDir (not just lumo.exe) so
# the sibling templates/plugins/builtin directories the binary depends
# on today land alongside it — see docs/architecture/distribution-protocol.md's
# "Why the whole archive". Chocolatey's shim generator auto-detects
# lumo.exe in $toolsDir and puts it on PATH; no explicit shim call
# needed.

$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition

$packageArgs = @{
  PackageName  = 'lumo-cli'
  UnzipLocation = $toolsDir
  Url          = 'https://github.com/intruder0007/Lumo/releases/download/v0.4.0/lumo_v0.4.0_windows_amd64.zip'
  Checksum     = '4157CBD2AC60553DBFFC1462C795617D7045CA1AD42BB24302804A89655E7B72'
  ChecksumType = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
