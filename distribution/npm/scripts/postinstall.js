#!/usr/bin/env node
// Downloads the release archive matching this package's own version and
// the current platform, verifies it against SHA256SUMS.txt, extracts
// just the bootstrap binary into .bin/, and cleans up. Zero npm
// dependencies on purpose — matches templates/node-rest-api and
// plugins/builtin/git-init's existing "shell out to a system tool
// rather than add a dependency" pattern (here: the system `tar`, which
// ships on Windows 10+/macOS/Linux by default and — as bsdtar — can
// extract .zip as well as .tar.gz, so one extraction path covers every
// platform). See docs/architecture/distribution-protocol.md.

'use strict';

const fs = require('fs');
const https = require('https');
const crypto = require('crypto');
const os = require('os');
const path = require('path');
const { execFileSync } = require('child_process');

const REPO = 'intruder0007/Cli';
const pkg = require('../package.json');
const VERSION = 'v' + pkg.version;

function platformTarget() {
  const goos = { linux: 'linux', darwin: 'darwin', win32: 'windows' }[process.platform];
  const goarch = { x64: 'amd64', arm64: 'arm64' }[process.arch];
  if (!goos || !goarch) {
    throw new Error(`unsupported platform: ${process.platform}/${process.arch}`);
  }
  return { goos, goarch };
}

function download(url, destPath) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(destPath);
    https
      .get(url, { headers: { 'User-Agent': 'bootstrap-cli-npm-installer' } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          file.close();
          fs.unlinkSync(destPath);
          download(res.headers.location, destPath).then(resolve, reject);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`GET ${url} failed: HTTP ${res.statusCode}`));
          return;
        }
        res.pipe(file);
        file.on('finish', () => file.close(resolve));
      })
      .on('error', reject);
  });
}

function sha256File(filePath) {
  const hash = crypto.createHash('sha256');
  hash.update(fs.readFileSync(filePath));
  return hash.digest('hex');
}

function verifyChecksum(archiveName, archivePath, sumsPath) {
  const sums = fs.readFileSync(sumsPath, 'utf8');
  const line = sums.split('\n').find((l) => l.trim().endsWith(archiveName));
  if (!line) {
    throw new Error(`no checksum entry for ${archiveName} in SHA256SUMS.txt`);
  }
  const expected = line.trim().split(/\s+/)[0];
  const actual = sha256File(archivePath);
  if (expected !== actual) {
    throw new Error(`checksum mismatch for ${archiveName}: expected ${expected}, got ${actual}`);
  }
}

async function main() {
  const { goos, goarch } = platformTarget();
  const ext = goos === 'windows' ? 'zip' : 'tar.gz';
  const archiveName = `cli_${VERSION}_${goos}_${goarch}.${ext}`;
  const baseUrl = `https://github.com/${REPO}/releases/download/${VERSION}`;

  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'bootstrap-cli-'));
  const archivePath = path.join(tmpDir, archiveName);
  const sumsPath = path.join(tmpDir, 'SHA256SUMS.txt');

  console.log(`bootstrap-cli: downloading ${archiveName} (${VERSION})...`);
  await download(`${baseUrl}/${archiveName}`, archivePath);
  await download(`${baseUrl}/SHA256SUMS.txt`, sumsPath);

  console.log('bootstrap-cli: verifying checksum...');
  verifyChecksum(archiveName, archivePath, sumsPath);

  console.log('bootstrap-cli: extracting...');
  // On Windows, a `tar` resolved via PATH can be Git for Windows' GNU
  // tar (from MSYS) rather than the OS-bundled bsdtar in System32 —
  // GNU tar interprets a bare drive-letter path like "C:\Users\..." as
  // remote-host syntax ("Cannot connect to C: resolve failed"), so the
  // real bsdtar is invoked explicitly by full path instead of trusting
  // PATH order. bsdtar (unlike GNU tar) also natively handles .zip, so
  // one call covers every platform's archive format.
  const tarPath =
    process.platform === 'win32'
      ? path.join(process.env.SystemRoot || 'C:\\Windows', 'System32', 'tar.exe')
      : 'tar';
  execFileSync(tarPath, ['-xf', archivePath, '-C', tmpDir]);

  const extractedDir = path.join(tmpDir, `cli_${VERSION}_${goos}_${goarch}`);
  const binName = goos === 'windows' ? 'bootstrap.exe' : 'bootstrap';
  const destDir = path.join(__dirname, '..', '.bin');
  fs.mkdirSync(destDir, { recursive: true });
  fs.copyFileSync(path.join(extractedDir, binName), path.join(destDir, binName));
  if (goos !== 'windows') {
    fs.chmodSync(path.join(destDir, binName), 0o755);
  }

  fs.rmSync(tmpDir, { recursive: true, force: true });
  console.log(`bootstrap-cli: installed ${VERSION} for ${goos}/${goarch}.`);
}

main().catch((err) => {
  console.error('bootstrap-cli: install failed:', err.message);
  process.exit(1);
});
