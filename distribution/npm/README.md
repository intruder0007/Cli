# npm wrapper — built and verified, not published

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).

`package.json` + a JS `bin` shim (`bin/bootstrap.js`) + a zero-dependency
`postinstall` script (`scripts/postinstall.js`) that downloads the
release archive matching this package's own version and the current
platform, verifies it against `SHA256SUMS.txt`, extracts just the
`bootstrap` binary via the system `tar` (bsdtar, invoked by full path on
Windows to avoid resolving to Git for Windows' GNU tar — see the comment
in `postinstall.js`), and places it in `.bin/`. `bin/bootstrap.js`
`spawnSync`s it with `stdio: 'inherit'` and forwards argv/exit code
exactly, per the protocol.

**Verified**: `npm pack` + `npm install --prefix <scratch>` against a
real, published release — the postinstall script really downloaded and
checksum-verified `v0.2.0`'s Windows archive, and the installed
`bootstrap` command ran correctly through the shim (`bootstrap version`
printed the right output).

**Not published.** `npm publish` needs a real npm auth token, which
isn't available in the environment that built this. Package name
(`bootstrap-cli`) is a placeholder, not reserved or claimed anywhere.
