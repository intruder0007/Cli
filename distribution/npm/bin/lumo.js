#!/usr/bin/env node
// Thin launcher only, per docs/architecture/distribution-protocol.md:
// never parses lumo's flags, never renders prompts. Spawns the
// real binary (postinstall.js placed it in ../.bin/) with stdio
// inherited (required for the interactive wizard's raw-mode arrow-key
// UI, ADR-0007) and forwards argv/exit code exactly.

'use strict';

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const binName = process.platform === 'win32' ? 'lumo.exe' : 'lumo';
const binPath = path.join(__dirname, '..', '.bin', binName);

if (!fs.existsSync(binPath)) {
  // The postinstall script downloads the binary; this path means it
  // never ran (npm install --ignore-scripts, --no-optional, or a
  // package manager that skips lifecycle scripts).
  console.error(
    `lumo-cli: the lumo binary is missing at ${binPath}.` +
      '\nIt is downloaded automatically by this package\'s postinstall script during' +
      '\n`npm install`. If you installed with --ignore-scripts or --no-optional,' +
      '\nre-run the install without those flags (e.g. `npm install --prefix "$(npm root -g)" lumo-cli`).'
  );
  process.exit(1);
}

const result = spawnSync(binPath, process.argv.slice(2), { stdio: 'inherit' });

if (result.error) {
  console.error(`lumo-cli: failed to launch ${binPath}: ${result.error.message}`);
  process.exit(1);
}
process.exit(result.status === null ? 1 : result.status);
