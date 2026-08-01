# npm wrapper (not implemented)

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).

Standard pattern for Go CLIs on npm: a thin JS `bin` shim (`postinstall`
downloads+verifies the matching release archive into the package's
install dir, or ship per-platform `optionalDependencies` packages each
bundling one archive) that `child_process.spawn`s the extracted
`bootstrap` with `stdio: 'inherit'` (required for the raw-mode wizard —
see the protocol doc's step 3) and forwards `process.argv.slice(2)` and
the exit code exactly.
