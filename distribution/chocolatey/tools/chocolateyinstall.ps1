$ErrorActionPreference = 'Stop'

# Extracts the whole archive into $toolsDir (not just bootstrap.exe) so
# the sibling templates/plugins/builtin directories the binary depends
# on today land alongside it — see docs/architecture/distribution-protocol.md's
# "Why the whole archive". Chocolatey's shim generator auto-detects
# bootstrap.exe in $toolsDir and puts it on PATH; no explicit shim call
# needed.

$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition

$packageArgs = @{
  PackageName  = 'bootstrap-cli'
  UnzipLocation = $toolsDir
  Url          = 'https://github.com/intruder0007/Cli/releases/download/v0.2.0/cli_v0.2.0_windows_amd64.zip'
  Checksum     = 'E0AA396C5EA19B13BF1929E4D6B94D576366FE76CF1A32E870199DB9FFBC732E'
  ChecksumType = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
