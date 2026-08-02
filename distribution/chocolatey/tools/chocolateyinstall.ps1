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
  Url          = 'https://github.com/intruder0007/Lumo/releases/download/v0.3.0/lumo_v0.3.0_windows_amd64.zip'
  Checksum     = '05CE82ABDE6942CA3236B41EE8C39EABFD483C16FF0B1125026E5DF2FD53D427'
  ChecksumType = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
