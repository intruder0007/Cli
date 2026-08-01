#!/usr/bin/env node
// Thin launcher only, per docs/architecture/distribution-protocol.md:
// never parses bootstrap's flags, never renders prompts. Spawns the
// real binary (postinstall.js placed it in ../.bin/) with stdio
// inherited (required for the interactive wizard's raw-mode arrow-key
// UI, ADR-0007) and forwards argv/exit code exactly.

'use strict';

const path = require('path');
const { spawnSync } = require('child_process');

const binName = process.platform === 'win32' ? 'bootstrap.exe' : 'bootstrap';
const binPath = path.join(__dirname, '..', '.bin', binName);

const result = spawnSync(binPath, process.argv.slice(2), { stdio: 'inherit' });

if (result.error) {
  console.error(`bootstrap-cli: failed to launch ${binPath}: ${result.error.message}`);
  process.exit(1);
}
process.exit(result.status === null ? 1 : result.status);
