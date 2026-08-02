#!/usr/bin/env node
// Pre-publish guard (package.json's prepublishOnly): proves that the
// GitHub release this package's own version points at actually exists
// with every archive the postinstall script needs — so `npm publish`
// cannot ship a wrapper whose install would fail for any supported
// platform. Runs on every `npm publish`, local or CI. Zero npm
// dependencies; exits non-zero with a message explaining the fix.

'use strict';

const https = require('https');

const REPO = 'intruder0007/Lumo';
const pkg = require('../package.json');
const VERSION = 'v' + pkg.version;

// Must mirror release.yml's matrix and postinstall.js's platformTarget.
const TARGETS = [
  { goos: 'linux', goarch: 'amd64', ext: 'tar.gz' },
  { goos: 'linux', goarch: 'arm64', ext: 'tar.gz' },
  { goos: 'darwin', goarch: 'amd64', ext: 'tar.gz' },
  { goos: 'darwin', goarch: 'arm64', ext: 'tar.gz' },
  { goos: 'windows', goarch: 'amd64', ext: 'zip' },
];

function headStatus(url) {
  return new Promise((resolve, reject) => {
    const req = https.request(
      url,
      { method: 'HEAD', headers: { 'User-Agent': 'lumo-cli-verify' } },
      (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          resolve(headStatus(res.headers.location));
          return;
        }
        res.resume();
        resolve(res.statusCode);
      }
    );
    req.on('error', reject);
    req.end();
  });
}

async function main() {
  const baseUrl = `https://github.com/${REPO}/releases/download/${VERSION}`;
  const urls = [
    ...TARGETS.map((t) => `${baseUrl}/lumo_${VERSION}_${t.goos}_${t.goarch}.${t.ext}`),
    `${baseUrl}/SHA256SUMS.txt`,
  ];

  const missing = [];
  for (const url of urls) {
    const status = await headStatus(url);
    if (status !== 200) {
      missing.push(`${url} (HTTP ${status})`);
    }
  }

  if (missing.length > 0) {
    console.error(
      `lumo-cli: cannot publish ${pkg.version} — ` +
        `the GitHub release does not have all assets the postinstall script needs:\n` +
        missing.map((m) => `  - ${m}`).join('\n') +
        `\nFix: cut the GitHub release first (tag v${pkg.version} via ` +
        `.github/workflows/release.yml) or bump package.json's version to an ` +
        `existing release, then publish again.`
    );
    process.exit(1);
  }

  console.log(
    `lumo-cli: verified ${TARGETS.length + 1} release assets for ${VERSION}.`
  );
}

main().catch((err) => {
  console.error(`lumo-cli: verification failed: ${err.message}`);
  process.exit(1);
});
