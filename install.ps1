# Installs the lumo CLI (github.com/intruder0007/Lumo) on Windows:
# downloads the matching release archive + SHA256SUMS.txt, verifies the
# checksum, and installs just lumo.exe onto PATH. See
# docs/architecture/adr/0012-universal-install-architecture.md — the
# binary is self-sufficient (embeds its own plugin set as a fallback),
# so this script never needs to preserve sibling directories the way a
# package-manager formula extracting the whole archive does.
#
# Usage:
#   iwr https://raw.githubusercontent.com/intruder0007/Lumo/main/install.ps1 -useb | iex
#   $env:VERSION = "v0.2.0"; iwr ... | iex   # pin a version

$ErrorActionPreference = "Stop"

$Repo = "intruder0007/Lumo"
$Version = if ($env:VERSION) { $env:VERSION } else { "latest" }
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { "$env:LOCALAPPDATA\Programs\lumo" }

if ($Version -eq "latest") {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $release.tag_name
    if (-not $Version) { throw "could not resolve the latest release version" }
}

$Arch = "amd64"
$Archive = "cli_${Version}_windows_${Arch}.zip"
$BaseUrl = "https://github.com/$Repo/releases/download/$Version"

$TmpDir = Join-Path $env:TEMP ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $TmpDir | Out-Null
try {
    Write-Host "Downloading $Archive ($Version)..."
    Invoke-WebRequest -Uri "$BaseUrl/$Archive" -OutFile "$TmpDir\$Archive" -UseBasicParsing
    Invoke-WebRequest -Uri "$BaseUrl/SHA256SUMS.txt" -OutFile "$TmpDir\SHA256SUMS.txt" -UseBasicParsing

    Write-Host "Verifying checksum..."
    $expectedLine = Select-String -Path "$TmpDir\SHA256SUMS.txt" -Pattern ([regex]::Escape($Archive))
    if (-not $expectedLine) { throw "no checksum entry found for $Archive in SHA256SUMS.txt" }
    $expected = ($expectedLine.Line -split '\s+')[0]
    $actual = (Get-FileHash -Path "$TmpDir\$Archive" -Algorithm SHA256).Hash.ToLower()
    if ($actual -ne $expected) { throw "checksum mismatch for $Archive (expected $expected, got $actual)" }

    Expand-Archive -Path "$TmpDir\$Archive" -DestinationPath $TmpDir -Force
    $extractedDir = Join-Path $TmpDir "cli_${Version}_windows_${Arch}"

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path (Join-Path $extractedDir "lumo.exe") -Destination (Join-Path $InstallDir "lumo.exe") -Force

    Write-Host "Installed lumo $Version to $InstallDir\lumo.exe"

    $onPath = ($env:Path -split ";") -contains $InstallDir
    if ($onPath) {
        Write-Host "Run 'lumo new' to get started."
    } else {
        Write-Host "$InstallDir is not on your PATH. Add it, e.g.:"
        Write-Host "  [Environment]::SetEnvironmentVariable('Path', `"`$env:Path;$InstallDir`", 'User')"
        Write-Host "then restart your terminal."
    }
} finally {
    Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
